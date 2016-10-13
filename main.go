package main

import (
	"flag"

	"github.com/RackHD/voyager-houston/server"
	log "github.com/sirupsen/logrus"
)

var (
	uri          = flag.String("uri", "amqp://guest:guest@rabbitmq:5672/", "AMQP URI")
	exchange     = flag.String("exchange", "Houston", "Durable, non-auto-deleted AMQP exchange name")
	exchangeType = flag.String("exchange-type", "topic", "Exchange type - direct|fanout|topic|x-custom")
	queue        = flag.String("queue", "voyager-houston-queue", "Ephemeral AMQP queue name")
	bindingKey   = flag.String("key", "test-key", "AMQP binding key")
	consumerTag  = flag.String("consumer-tag", "simple-consumer", "AMQP consumer tag (should not be blank)")
	dbAddress    = flag.String("db-address", "root@(mysql:3306)/mysql", "Address of the MySQL database")
)

func init() {
	flag.Parse()
}

func main() {
	s := server.NewServer(*uri, *dbAddress)
	defer s.MQ.Close()

	log.Info("Trying to init IPAM now")
	s.InitIPAM()

	// Listen for messages in the background in infinite loop
	s.ListenOnHoustonReceiveQueue()
	s.Run()

}
