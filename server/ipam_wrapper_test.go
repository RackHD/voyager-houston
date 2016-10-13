package server_test

import (
	"encoding/json"
	"fmt"
	"time"
	"io/ioutil"
	"net/http"

	log "github.com/sirupsen/logrus"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/RackHD/voyager-houston/server"
	"github.com/RackHD/voyager-utilities/models"
)

var _ = Describe("Houston interface to IPAM", func() {
	Context("Houston can Initialize IPAM", func() {

		var uri string
		var dbAddress string
		var s *server.Server

		BeforeEach(func() {
			uri = "amqp://guest:guest@localhost:5672/"
			dbAddress = "root@(localhost:3306)/mysql"
		})

		AfterEach(func() {
			s.MySQL.DB.DropTableIfExists("pool_entities")
			s.MySQL.DB.DropTableIfExists("subnet_entities")
		})

		It("INTEGRATION should initialize IPAM with one pool and one subnet", func() {
			s = server.NewServer(uri, dbAddress)
			defer s.MQ.Close()
			s.InitIPAM()

			testPool := models.PoolEntity{}
			testSubnet := models.SubnetEntity{}

			// Get the new Pool, verify ID was assigned and name is correct
			err := s.MySQL.DB.Find(&testPool).Error
			Expect(err).ToNot(HaveOccurred())
			Expect(testPool.ID).ToNot(Equal(""))
			Expect(testPool.Name).To(Equal(models.DefaultKey))

			// Get new subnet, verify ID was assigned, name is correct, and is associated with pool
			err = s.MySQL.DB.Where("pool_id = ?", testPool.ID).Find(&testSubnet).Error
			Expect(err).ToNot(HaveOccurred())
			Expect(testSubnet.ID).ToNot(Equal(""))
			Expect(testSubnet.Name).To(Equal(models.DefaultKey))
			Expect(testSubnet.PoolID).To(Equal(testPool.ID))

			// Validate the Pool was created in IPAM
			poolURL := fmt.Sprintf("http://localhost:8000/pools/%s", testPool.ID)
			poolResp, err := http.Get(poolURL)
			Expect(err).ToNot(HaveOccurred())
			Expect(poolResp.StatusCode).To(Equal(200))
			defer poolResp.Body.Close()

			// Parse response
			var ipamPoolMsg interface{}
			poolBody, err := ioutil.ReadAll(poolResp.Body)
			Expect(err).ToNot(HaveOccurred())
			err = json.Unmarshal(poolBody, &ipamPoolMsg)
			Expect(err).ToNot(HaveOccurred())
			ipamPool := ipamPoolMsg.(map[string]interface{})
			log.Infof("ipamPoolMsg is %+v", ipamPool["name"])
			Expect(ipamPool["name"]).To(Equal(models.DefaultKey))

			// Validate the subnet was created in IPAM
			subnetURL := fmt.Sprintf("http://localhost:8000/subnets/%s", testSubnet.ID)
			subnetResp, err := http.Get(subnetURL)
			Expect(err).ToNot(HaveOccurred())
			Expect(subnetResp.StatusCode).To(Equal(200))
			defer subnetResp.Body.Close()

			// Parse response
			var ipamSubnetMsg interface{}
			subnetBody, err := ioutil.ReadAll(subnetResp.Body)
			Expect(err).ToNot(HaveOccurred())
			err = json.Unmarshal(subnetBody, &ipamSubnetMsg)
			Expect(err).ToNot(HaveOccurred())
			ipamSubnet := ipamSubnetMsg.(map[string]interface{})
			log.Infof("ipamSubnetMsg is %+v", ipamSubnet["name"])
			Expect(ipamSubnet["name"]).To(Equal(models.DefaultKey))
			Expect(ipamSubnet["pool"]).To(Equal(testPool.ID))

		})

		It("INTEGRATION should not initialize IPAM if it is already initialized.", func() {
			s = server.NewServer(uri, dbAddress)
			defer s.MQ.Close()
			s.InitIPAM()
			time.Sleep(time.Second * 5)
			s.InitIPAM()

			testPools := []models.PoolEntity{}
			err := s.MySQL.DB.Find(&testPools).Error
			Expect(err).ToNot(HaveOccurred())
			Expect(len(testPools)).To(Equal(1))

			testSubnets := []models.SubnetEntity{}
			err = s.MySQL.DB.Find(&testSubnets).Error
			Expect(err).ToNot(HaveOccurred())
			Expect(len(testSubnets)).To(Equal(1))
		})

	})

})
