package server

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/RackHD/voyager-utilities/models"
	"github.com/RackHD/voyager-utilities/random"
	log "github.com/sirupsen/logrus"
)

// InitIPAM is ...
func (s *Server) InitIPAM() {

	// Check if IPAM is already initialized
	ipamIsInit, err := s.isIpamInit()
	if err != nil {
		if err != nil {
			log.Fatalf("Houston could not determine IPAM status: %s", err)
		}
	}

	if !ipamIsInit {
		// Create default pools. Configurable in the future
		poolID, err := s.CreatePool(models.DefaultKey, models.DefaultRange)
		if err != nil {
			log.Fatalf("Could not initialize Houston's connection to IPAM: %s", err)
		}
		log.Info("Initialized IPAM with pool: ", poolID)

		// Create default subnets. Configurable in the future
		subnetID, err := s.CreateSubnet(models.DefaultKey, poolID, models.DefaultStart, models.DefaultEnd)
		if err != nil {
			log.Fatalf("Could not initialize Houston's connection to IPAM: %s", err)
		}
		log.Info("Initialized IPAM with subnet: ", subnetID)

	} else {
		log.Info("IPAM is already initialized")
	}

}

// CreatePool creates a pool in IPAM
func (s *Server) CreatePool(name, metadata string) (string, error) {
	queueName := random.RandQueue()
	correlationID := random.RandQueue()
	consumerTag := random.RandQueue()
	finish := make(chan struct{})
	poolID := ""

	//Create pool
	message, err := json.Marshal(models.IPAMPoolMsg{
		Name:       name,
		Action:     models.CreateAction,
		ObjectType: models.PoolType,
		Metadata:   metadata,
	})

	if err != nil {
		log.Info("Error: ", err, "")
		return "", err
	}

	// Listen
	mqChannel, deliveries, err := s.MQ.Listen(models.IpamExchange, models.IpamExchangeType, queueName, models.HoustonRepliesQueue, consumerTag)
	if err != nil {
		log.Info("Error listening: %s", err)
		return "", err
	}

	defer func() {
		log.Info("Trying to close the channel")
		if err = mqChannel.Cancel(consumerTag, false); err != nil {
			log.Info("Consumer cancel failed: %s", err)
		}

		log.Info("Trying to delete the queue: ", queueName)
		if _, err = mqChannel.QueueDelete(queueName, false, false, true); err != nil {
			log.Info("Queue Delete failed: %s", err)
		}

		if err = mqChannel.Close(); err != nil {
			log.Info("Closing channel failed: ", err)
		}
		close(finish)
	}()

	err = s.MQ.Send(models.IpamExchange, models.IpamExchangeType, models.IpamReceiveQueue, string(message), correlationID, models.HoustonRepliesQueue)

	if err != nil {
		log.Info("Error: ", err, "")
		return "", err
	}

	// poll the listener for response with matching correlationID
	log.Info("Listening to deliveries (pool)")
	cancel := make(chan struct{})
	defer close(cancel)
	go func() {
		// Write to finish channel before exiting Gofunc
		defer func() {
			finish <- struct{}{}
		}()

		for {
			select {
			case d := <-deliveries:
				log.Info("Got a message")
				if d.CorrelationId == correlationID {

					log.Info("Pool message: ", string(d.Body))

					// Extract PoolID from d.Body
					reply := models.IPAMPoolMsg{}
					err = json.Unmarshal(d.Body, &reply)
					if err != nil {
						log.Warnf("Error unmarshaling voyager-ipam-service response %s", err)
					}

					newPool := models.PoolEntity{
						ID:   reply.ID,
						Name: reply.Name,
					}

					log.Info("Adding new pool to DB: ", newPool)
					err = s.MySQL.DB.Create(&newPool).Error
					log.Info("I tried to add it")
					if err != nil {
						log.Info("Error: ", err)
						poolID = ""
						return
					}

					log.Info("Created Pool: ", newPool.ID)
					d.Ack(true)
					poolID = newPool.ID
					return
				}
			case <-cancel:
				poolID = ""
				return
			}
		}
		poolID = ""
		return
	}()

	// Time out after 5 seconds
	select {
	case <-time.After(5 * time.Second):
		cancel <- struct{}{} // send signal to cancel the goroutine!
		return "", fmt.Errorf("Error: Request to Create pool timed out after 5s")
	case <-finish:
		return poolID, nil
	}
}

// CreateSubnet creates a pool in IPAM
func (s *Server) CreateSubnet(name, poolID, start, end string) (string, error) {
	queueName := random.RandQueue()
	correlationID := random.RandQueue()
	consumerTag := random.RandQueue()
	finish := make(chan struct{})
	subnetID := ""

	//Create pool
	message, err := json.Marshal(models.IPAMSubnetMsg{
		Name:       name,
		Action:     models.CreateAction,
		ObjectType: models.SubnetType,
		Pool:       poolID,
		Start:      start,
		End:        end,
	})

	if err != nil {
		log.Info("Error: ", err)
		return "", err
	}

	// Listen
	mqChannel, deliveries, err := s.MQ.Listen(models.IpamExchange, models.IpamExchangeType, queueName, models.HoustonRepliesQueue, consumerTag)
	if err != nil {
		log.Info("Error listening: %s", err)
		return "", err
	}
	defer func() {
		log.Info("Trying to close the channel")
		if err = mqChannel.Cancel(consumerTag, false); err != nil {
			log.Info("Consumer cancel failed: %s", err)
		}

		log.Info("Trying to delete the queue: ", queueName)
		if _, err = mqChannel.QueueDelete(queueName, false, false, true); err != nil {
			log.Info("Queue Delete failed: %s", err)
		}

		if err = mqChannel.Close(); err != nil {
			log.Info("Closing channel failed: ", err)
		}
		close(finish)
	}()

	err = s.MQ.Send(models.IpamExchange, models.IpamExchangeType, models.IpamReceiveQueue, string(message), correlationID, models.HoustonRepliesQueue)

	if err != nil {
		log.Info("Error: ", err, "")
		return "", err
	}

	// poll the listener for response with matching correlationID
	log.Info("Listening to deliveries")
	cancel := make(chan struct{})
	defer close(cancel)
	go func() {
		// Write to finish channel before exiting Gofunc
		defer func() {
			finish <- struct{}{}
		}()

		for {
			select {
			case d := <-deliveries:
				log.Info("Got a message")
				if d.CorrelationId == correlationID {
					log.Info("Subnet message: ", string(d.Body))

					// Extract PoolID from d.Body
					reply := models.IPAMSubnetMsg{}
					err = json.Unmarshal(d.Body, &reply)
					if err != nil {
						log.Warnf("Error unmarshaling voyager-ipam-service response %s", err)
					}

					newSubnet := models.SubnetEntity{
						ID:     reply.ID,
						Name:   reply.Name,
						PoolID: reply.Pool,
					}

					log.Info("Adding new subnet to DB: ", newSubnet)
					if err = s.MySQL.DB.Create(&newSubnet).Error; err != nil {
						log.Infof("Error: %s", err)
						subnetID = ""
					}

					d.Ack(true)
					log.Infof("Created Subnet: %s", newSubnet.ID)
					subnetID = newSubnet.ID
					return
				}
			case <-cancel:
				subnetID = ""
				return
			}
		}
		subnetID = ""
		return
	}()

	// Time out after 5 seconds
	select {
	case <-time.After(5 * time.Second):
		cancel <- struct{}{} // send signal to cancel the goroutine!
		return "", fmt.Errorf("Error: Request to Create subnet timed out after 5s")
	case <-finish:
		return subnetID, nil
	}

}

// returns whether IPAM has been initialized, or false with an error.
func (s *Server) isIpamInit() (bool, error) {
	pools := []models.PoolEntity{}

	if hasPools := s.MySQL.DB.HasTable(&models.PoolEntity{}); hasPools {

		if err := s.MySQL.DB.Where("Name = ?", models.DefaultKey).Find(&pools).Error; err != nil {
			log.Info("Error accessing DB: ", err)
			return false, err
		}

		for _, pool := range pools {
			if pool.Name == models.DefaultKey {
				return true, nil
			}
		}

		log.Info("Table exists but Pools is empty")
		// Pools table exists but ours isn't in it.
		return false, nil

	}
	log.Info("No table in DB")
	// Pools table does not exist
	return false, nil
}
