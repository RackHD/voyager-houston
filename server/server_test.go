package server_test

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/RackHD/voyager-houston/model"
	. "github.com/RackHD/voyager-houston/server"
	"github.com/RackHD/voyager-utilities/random"
	"github.com/streadway/amqp"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Server", func() {
	var rabbitMQURL string
	var dbAddress string
	var s *Server

	BeforeEach(func() {
		rabbitMQURL = "amqp://guest:guest@localhost:5672/"
		dbAddress = "root@(localhost:3306)/mysql"
	})
	Context("API endpoints", func() {
		It("UNIT should return default name", func() {
			os.Setenv("PORT", "8080")
			serverURL := "http://localhost:8080"
			os.Chdir("..")
			s := Server{}

			devNull, err := os.Open(os.DevNull)
			if err != nil {
				log.Panic("Unable to open ", os.DevNull, err)
			}

			gin.SetMode(gin.ReleaseMode)
			gin.DefaultWriter = devNull
			gin.LoggerWithWriter(ioutil.Discard)

			go s.Run()
			time.Sleep(time.Millisecond * 500)

			req, err := http.NewRequest("GET", serverURL+"/info", nil)
			Expect(err).ToNot(HaveOccurred())
			resp, err := (&http.Client{}).Do(req)
			Expect(err).ToNot(HaveOccurred())
			defer resp.Body.Close()

			body, err := ioutil.ReadAll(resp.Body)
			Expect(err).ToNot(HaveOccurred())

			var voyager-houston model.Houston
			err = json.Unmarshal(body, &voyager-houston)
			Expect(err).ToNot(HaveOccurred())
			Expect(voyager-houston.Name).To(Equal("voyager"))
		})
		Context("When handling AMQP messages", func() {
			BeforeEach(func() {
				s = NewServer(rabbitMQURL, dbAddress)
				Expect(s).ToNot(Equal(nil))
			})
			It("INTEGRATION nodes should timeout after 5 seconds if no response", func() {
				os.Setenv("PORT", "8080")
				serverURL := "http://localhost:8080"
				os.Chdir("..")

				devNull, err := os.Open(os.DevNull)
				if err != nil {
					log.Panic("Unable to open ", os.DevNull, err)
				}

				gin.SetMode(gin.ReleaseMode)
				gin.DefaultWriter = devNull
				gin.LoggerWithWriter(ioutil.Discard)

				go s.Run()
				time.Sleep(time.Millisecond * 500)

				start := time.Now()
				req, err := http.NewRequest("GET", serverURL+"/nodes", nil)
				Expect(err).ToNot(HaveOccurred())
				resp, _ := (&http.Client{}).Do(req)
				end := time.Now()

				Expect(end.Sub(start).Seconds()).To(BeNumerically(">=", 5))
				Expect(resp.StatusCode).To(Equal(504))
			})

			It("INTEGRATION nodes API should send RabbitMQ request to 'get_nodes'", func() {
				var requests <-chan amqp.Delivery
				inventoryServiceExchange := "voyager-inventory-service"
				inventoryServiceExchangeType := "topic"
				inventoryServiceRoutingKey := "requests"
				inventoryServiceQueueName := random.RandQueue()
				inventoryServiceConsumerTag := ""

				// Initialize test server
				os.Setenv("PORT", "8080")
				serverURL := "http://localhost:8080"
				os.Chdir("..")
				devNull, err := os.Open(os.DevNull)
				if err != nil {
					log.Panic("Unable to open ", os.DevNull, err)
				}

				gin.SetMode(gin.ReleaseMode)
				gin.DefaultWriter = devNull
				gin.LoggerWithWriter(ioutil.Discard)
				go s.Run()
				time.Sleep(time.Millisecond * 500)

				// Set up listener for server's outgoing RMQ message
				_, requests, err = s.MQ.Listen(inventoryServiceExchange, inventoryServiceExchangeType, inventoryServiceQueueName, inventoryServiceRoutingKey, inventoryServiceConsumerTag)
				Expect(err).ToNot(HaveOccurred())

				// Call Houston's 	'/nodes' API
				req, err := http.NewRequest("GET", serverURL+"/nodes", nil)
				Expect(err).ToNot(HaveOccurred())
				_, err = (&http.Client{}).Do(req)
				Expect(err).ToNot(HaveOccurred())

				// Intercept Houston's outgoing RMQ call to voyager-inventory-service and validate it
				out := <-requests

				out.Ack(false)
				Expect(string(out.Body)).To(Equal(`{"command": "get_nodes", "options":""}`))
			})
		})

	})
})
