package main

import (
	"ardiansyah3ber/wa-biz/models"
	"ardiansyah3ber/wa-biz/postgres"
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
)

type Customer struct {
	Name   string `json:"name"`
	Phone  string `json:"phone"`
	ItemID string `json:"item"`
}

var (
	// ID GROUP UNTUK FJB NYA
	groupID = "120363273961691637"

	itemChatCaption = make(map[string]string)     // ITEM CHAT UNTUK MENYIMPAN CAPTION GAMBAR, DENGAN KEY ID CHAT
	customerChat    = make(map[string]Customer)   // MENYIMPAN REPLY CUSTOMER BESERTA DATA CUSTOMER DAN ITEM ID YANG DI REPLY
	reactConfirmed  = make(map[string][]Customer) // ARRAY CUSTOMER YANG DI KONFIRMASI BEDASARKAN ITEM ID
)

func contains(arr []string, value string) bool {
	for _, v := range arr {
		if v == value {
			return true
		}
	}
	return false
}

func eventHandler(db *sqlx.DB) func(interface{}) {
	words := []string{"fix", "mau"}
	return func(evt interface{}) {
		switch v := evt.(type) {
		case *events.Message:
			if v.Info.MessageSource.Chat.User == groupID {
				if v.Info.IsFromMe {
					img := v.Message.GetImageMessage()
					itemChatID := v.Info.ID // ID CHAT YANG DIKIRIM ISTRINYA ARDI
					if img != nil {
						itemChatCaption[itemChatID] = img.GetCaption()

						// SAVE DB PRODUCT
						caption := img.GetCaption()
						splitCaption := strings.Split(caption, " ")

						removeLast := splitCaption[:len(splitCaption)-1]
						title := strings.Join(removeLast, " ")
						priceString := strings.Replace(splitCaption[len(splitCaption)-1], "K", "000", -1)
						priceString = strings.Replace(priceString, "k", "000", -1)
						price, err := strconv.Atoi(priceString)
						if err != nil {
							log.Fatal(err)
						}

						tx := db.MustBegin()
						tx.MustExec("insert into wabiz_products (title, price, message_id) values ($1, $2, $3)", title, price, itemChatID)
						tx.Commit()
					}

					reactMsg := v.Message.GetReactionMessage()
					if reactMsg != nil {
						msgId := reactMsg.GetKey().GetId()
						if reactMsg.GetText() == "❤️" {
							tx := db.MustBegin()
							tx.MustExec("UPDATE wabiz_replies SET status = $1 WHERE message_id = $2", "YES", msgId)
							tx.Commit()
						}
					}

				} else {
					ext := v.Message.GetExtendedTextMessage()
					if ext != nil {
						if ext.Text != nil {
							t := strings.ToLower(ext.GetText()) // CHECK FIX/MAU/IYA/DSB
							if contains(words, t) {
								if ext.ContextInfo.QuotedMessage.ImageMessage != nil {
									custChatID := v.Info.ID
									repliedID := ext.ContextInfo.GetStanzaId()
									fmt.Printf("Ada reply dari %s %s\n", v.Info.PushName, v.Info.Sender.User)
									customerChat[custChatID] = Customer{
										Name:   v.Info.PushName,
										Phone:  v.Info.Sender.User,
										ItemID: repliedID,
									}

									tx := db.MustBegin()
									tx.MustExec("insert into wabiz_replies (name, phone, message_id, status, product_id) values ($1, $2, $3, $4, $5)", v.Info.PushName, v.Info.Sender.User, custChatID, "WAIT", repliedID)
									tx.Commit()
								}
							}
						}
					}
				}
			}
		}
	}
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func GetProducts(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.DB.Query("select * from wabiz_products")
		if err != nil {
			log.Fatal(err)
		}

		var products []models.Product

		for rows.Next() {
			var p models.Product
			if err := rows.Scan(&p.ID, &p.CreatedAt, &p.Title, &p.Price, &p.MessageID); err != nil {
				log.Fatal(err)
			}
			products = append(products, p)
		}

		json.NewEncoder(w).Encode(products)
	}
}

func GetReplies(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.DB.Query("select * from wabiz_replies")
		if err != nil {
			log.Fatal(err)
		}

		var products []models.Reply

		for rows.Next() {
			var p models.Reply
			if err := rows.Scan(&p.ID, &p.CreatedAt, &p.Name, &p.Phone, &p.MessageID, &p.Status, &p.ProductID); err != nil {
				log.Fatal(err)
			}
			products = append(products, p)
		}

		json.NewEncoder(w).Encode(products)
	}
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	database_type := os.Getenv("DATABASE_TYPE")
	database_url := os.Getenv("DATABASE_URL")

	// db products
	db := postgres.ConnectDB(database_type, database_url)

	// db auth whatsapp
	dbLog := waLog.Stdout("Database", "DEBUG", true)
	container, err := sqlstore.New("sqlite3", "file:store.db?_foreign_keys=on", dbLog)
	if err != nil {
		panic(err)
	}

	deviceStore, err := container.GetFirstDevice()
	if err != nil {
		panic(err)
	}
	clientLog := waLog.Stdout("Client", "INFO", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)
	client.AddEventHandler(eventHandler(db))

	if client.Store.ID == nil {
		qrChan, _ := client.GetQRChannel(context.Background())
		err = client.Connect()
		if err != nil {
			panic(err)
		}
		for evt := range qrChan {
			if evt.Event == "code" {
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				fmt.Println("QR code:", evt.Code)
			} else {
				fmt.Println("Login event:", evt.Event)
			}
		}
	} else {
		err = client.Connect()
		if err != nil {
			panic(err)
		}
	}

	// serve endpoint
	r := mux.NewRouter()
	r.HandleFunc("/", HomeHandler).Methods("GET")
	r.HandleFunc("/products", GetProducts(db)).Methods("GET")
	r.HandleFunc("/replies", GetReplies(db)).Methods("GET")

	http.ListenAndServe("0.0.0.0:5000", r)

	defer db.Close()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	client.Disconnect()
}
