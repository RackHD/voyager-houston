package server

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/RackHD/voyager-houston/model"
	"github.com/RackHD/voyager-houston/mysql"
	"github.com/RackHD/voyager-utilities/amqp"
	"github.com/RackHD/voyager-utilities/models"
	"github.com/RackHD/voyager-utilities/random"
	log "github.com/sirupsen/logrus"
	samqp "github.com/streadway/amqp"
)

// Server is a Voyager server
type Server struct {
	MQ    *amqp.Client
	MySQL *mysql.DBconn
}

// NewServer connects to AMQP and returns the server object
func NewServer(amqpAddress, dbAddress string) *Server {
	rand.Seed(time.Now().UTC().UnixNano())

	server := Server{}
	server.MQ = amqp.NewClient(amqpAddress)
	if server.MQ == nil {
		log.Fatalf("Could not connect to RabbitMQ at %s\n", amqpAddress)
	}

	server.MySQL = &mysql.DBconn{}
	err := server.MySQL.Initialize(dbAddress)
	if err != nil {
		log.Fatalf("Error connecting to DB: %s\n", err)
	}

	return &server
}

// Run it
func (s *Server) Run() {

	port := os.Getenv("PORT")
	server := gin.Default()

	server.GET("/info", s.InfoHandler)
	server.GET("/nodes", s.NodesHandler)

	log.Info("Starting Voyager at Port ", port)
	server.Run(":" + port)

}

// InfoHandler Serves /info
func (s *Server) InfoHandler(c *gin.Context) {
	c.JSON(http.StatusOK, model.Houston{
		Name: "voyager",
	})
}

// NodesHandler Serves /nodes
func (s *Server) NodesHandler(c *gin.Context) {
	exchangeName := "voyager-inventory-service"
	exchangeType := "topic"
	queueName := random.RandQueue()
	receieveQueue := "replies"
	sendQueue := "requests"
	corID := random.RandQueue()
	consumerTag := random.RandQueue()
	requestMessage := `{"command": "get_nodes", "options":""}`
	finish := make(chan struct{})

	mqChannel, deliveries, err := s.MQ.Listen(exchangeName, exchangeType, queueName, receieveQueue, consumerTag)
	if err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf("%s", err))
		log.Info("Error listening: %s\n", err)
		c.JSON(500, "Error Listening")
		c.Abort()
		return
	}

	defer func() {
		log.Info("Trying to close the channel\n")
		if err = mqChannel.Cancel(consumerTag, false); err != nil {
			log.Info("Consumer cancel failed: %s\n", err)
		}

		log.Info("Trying to delete the queue: %s\n", queueName)
		if _, err = mqChannel.QueueDelete(queueName, false, false, true); err != nil {
			log.Info("Queue Delete failed: %s\n", err)
		}

		if err = mqChannel.Close(); err != nil {
			log.Info("Closing channel failed: ", err)
		}
		close(finish)
	}()

	go func() {
		for d := range deliveries {
			if d.CorrelationId == corID {

				d.Ack(true)
				c.String(http.StatusOK, string(d.Body))
				finish <- struct{}{}
				return
			}
		}
	}()

	err = s.MQ.Send(exchangeName, exchangeType, sendQueue, requestMessage, corID, receieveQueue)
	if err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf("%s", err))
		return
	}

	// Time out after 5 seconds
	select {
	case <-time.After(5 * time.Second):
		c.JSON(504, "Request Timed Out")
		c.Abort()
	case <-finish:
	}

}

// ListenOnHoustonReceiveQueue is ...
func (s *Server) ListenOnHoustonReceiveQueue() error {
	houstonQueueName := random.RandQueue()
	_, deliveries, err := s.MQ.Listen(models.HoustonExchange, models.HoustonExchangeType, houstonQueueName, models.HoustonReceiveQueue, "")
	if err != nil {
		return err
	}

	go func() {
		for m := range deliveries {
			log.Info("Got ", len(m.Body), "B delivery on exchange ", m.Exchange, ": ", string(m.Body))
			m.Ack(true)
			go s.ProcessAMQPMessage(&m)
		}
	}()

	return nil
}

// ProcessAMQPMessage is a handler for incoming AMQP messages
func (s *Server) ProcessAMQPMessage(m *samqp.Delivery) error {
	switch m.Exchange {

	case "Houston":
		log.Info("Not yet implemented")

	default:
		log.Warnf("Unknown exchange: %s\n", m.Exchange)

	}

	return nil
}
