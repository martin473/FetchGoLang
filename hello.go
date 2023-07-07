/*WEBSITE DESCRIPTION 
This code uses mux to handle incoming URL requests. It expects POST and GET to specific endpoints
and uses mux & http to handle the data to/from those endpoints

It uses encoding/json to decode json into structs for storage and parsing and encode responses into
http/json format to return to the user

It expects a POST request with a JSON receipt to the /receipts/process endpoint. It will generate an
ID for the receipt, store it, and return the ID. The receipt can then be accessed at the 
/receipts/{id}/points endpoint. It expects a GET request with the proper UUID for {id}. It will
access the stored receipt, calculate the points, and return the points to the user.

I unfortunately did not have time to write test code. Or seriously test the program besides
basic functionality. There are issues with point calculation*/

/*POINT CALCULATION PSEUDOCODE
points += Retailer name >> trim whitespace >> count isletter or is number
points += 50 if total == total.floor
points += 25 if total % 0.25 == 0
points += 5 * int(count of items / 2)
for each item
	if (item description >> trimwhitespace) % 3
		points += ceiling(itemprice * 0.2)
	if (purchase day >> use time lib to convert to int) % 2 != 0
		points += 6
	if purchase time > 14:00 OR purchase time < 16:00
		points += 10
return points

/*KNOWN ERRORS
String formatting/debugging - I am new to this in go so it's very messy. ideally it would pretty print
a point count in the browser similar to the example using stored variables based on input and calculated vals. 
This would allow faster debugging. Currently the point count is off. 

/*MY EXPERIENCE & METHOD 
I am pretty bad at providing environment setups, so I figured it would be easier to learn
Go since the env setup is so easy and your team didn't need extra instructions.

Learning go still took longer than expected, though it's pretty similar to C, Java, Python, Javascript.
It's also braindead easy to write a little webserver. So I will probably continue to use this language.

My method, since there was not a lot of time to learn all the language specific tools, was to use ChatGPT
for the first iteration of code. I pseudocoded the points logic as well as some of the general system flow.

ChatGPT, predictably, released code that was 50% there. From that structure I was able to read up on
the tools it used, general syntax and implementation, and figure out slightly more accurate applications
of what chatGPT provided

It did save me a lot of time searching for modules like UUID and Mux, as well as time looking up which
functions and boilerplate would be most effective.

A huge problem with this method is that I'm storing the receipts globally. Ideally I'd use a database.
Second to that I might store them in a function instance. However, because I'm so new with go and the
timeline is running a bit short, I coded it that way just to get something functional.

My goal as a Go developer is to move pretty far from chatGPT due to errors and copyright. Also, in general
it has a less detailed command of the current issues and needs with software, and it's usually faster to
develop on your own when you're fluent in a language. But it was a helpful way to explore the basics of the
language.
*/
package main

import (
	"encoding/json"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

//global hashmap of receipts and ids that is not threadsafe
//bad implementation
var receipts = make(map[string]Receipt)

//receipt struct
type Receipt struct {
	Retailer     string `json:"retailer"`
	PurchaseDate string `json:"purchaseDate"`
	PurchaseTime string `json:"purchaseTime"`
	Items        []struct {
		ShortDescription string `json:"shortDescription"`
		Price            string `json:"price"`
	} `json:"items"`
	Total string `json:"total"`
}

//main sets up router and port
func main() {
	router := setupRouter()
	log.Fatal(http.ListenAndServe(":8080", router))
}

//setup router creates new mux instance for managing endpoints
	//has post and get endpoints as specified by documentation
	//receipts/process expects a POST with valid receipt JSON. It returns a UUID as JSON 
		//and stores a receipt and UUID globally in a hashmap
	//receipts/{id}/points uses the mux to take ID from the URI string. It uses it to retreive the receipt from the
		//hashmap and then runs point calculator on the receipt. It then returns points as a JSON string
func setupRouter() *mux.Router {
	router := mux.NewRouter()
	router.HandleFunc("/receipts/process", ProcessReceipts).Methods("POST")
	router.HandleFunc("/receipts/{id}/points", GetPoints).Methods("GET")
	return router
}

//makes an http request
//if there's no error decoding the request json into a receipt
	//generates a UUID receiptID string
	//creates a struct response with the JSON ID and receipt ID
	//encodes the response as json and returns to user
func ProcessReceipts(w http.ResponseWriter, r *http.Request) {
	var receipt Receipt
	if err := json.NewDecoder(r.Body).Decode(&receipt); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	//generate UUID
	receiptID := generateReceiptID()
	//Stores receipt and receiptID as KV pair in a global hashmap
	receipts[receiptID] = receipt
	//return receiptID as JSON to user
	json.NewEncoder(w).Encode(receiptID)
}

//uses mux to get receiptID from URI string
	//no error checking for this currently
//runs calculatePoints based on global receipts variable
//creates a response struct with the points and label
//json encodes the response and returns it to the user
func GetPoints(w http.ResponseWriter, r *http.Request) {
	//decodes request params and finds ID
	params := mux.Vars(r)
	receiptID := params["id"]

	//should then find the receipt by ID and use calculate points on a receipt not a receiptID
	points := calculatePoints(receipts[receiptID])

	//creates a json response struct that has the points
	response := struct {
		Points int `json:"points"`
	}{
		Points: points,
	}

	//returns the points as json response
	json.NewEncoder(w).Encode(response)
}

//Generates UUID String for receipt
func generateReceiptID() string {
	return uuid.New().String()
}

// calculatePoints calculates the points earned for a receipt.
//haven't double checked this but it's close to my psuedocode
func calculatePoints(receipt Receipt) int {

	//1 point for each char (doesn't check for alphanumeric)
	points := len(strings.TrimSpace(receipt.Retailer))

	//Total is total string typecast to 64 bit float
	total, err := strconv.ParseFloat(receipt.Total, 64)
	if err != nil {
		log.Println("Failed to parse total:", err)
	}

	//If total is round
	if total == math.Floor(total) {
		points += 50
	}

	//if total is a multiple of 0.25
	if math.Mod(total, 0.25) == 0 {
		points += 25
	}

	//every two items on the receipt
	points += (len(receipt.Items) / 2) * 5

	//is checking to see if the description is divisible by 3
	//another parse float conversion
	//adds the ceiling of the price * 0.2
	for _, item := range receipt.Items {
		trimmedLength := len(strings.TrimSpace(item.ShortDescription))
		if trimmedLength%3 == 0 {
			price, err := strconv.ParseFloat(item.Price, 64)
			if err != nil {
				log.Println("Failed to parse item price:", err)
				continue
			}
			points += int(math.Ceil(price * 0.2))
		}
	}

//not sure why it's parsing "15:04" here
	purchaseTime, err := time.Parse("15:04", receipt.PurchaseTime)
	if err != nil {
		log.Println("Failed to parse purchase time:", err)
		return points
	}
	//should have purchase time parsed and then check to see when the hours are
	if purchaseTime.Hour() > 14 && purchaseTime.Hour() < 16 {
		points += 10
	}

	//running parse time again with an inline date? Maybe this is a format argument
	purchaseDate, err := time.Parse("2006-01-02", receipt.PurchaseDate)
	if err != nil {
		log.Println("Failed to parse purchase date:", err)
		return points
	}

	//checks for even day
	if purchaseDate.Day()%2 != 0 {
		points += 6
	}

	return points
}