package mysql

import (
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql" // Blank import because the library says to.
	"github.com/jinzhu/gorm"
	"github.com/RackHD/voyager-utilities/models"
)

// DBconn is a struct for maintaining a connection to the MySQL Database
type DBconn struct {
	DB *gorm.DB
}

// Initialize attempts to open and verify a connection to the DB
func (d *DBconn) Initialize(address string) error {
	var err error

	log.Printf("IM AN INIT FUNC\n\n\n")

	for i := 0; i < 18; i++ {
		d.DB, err = gorm.Open("mysql", address)
		if err != nil {
			log.Printf("Could not connect. Sleeping for 10s %s\n", err)
			time.Sleep(10 * time.Second)
			continue
		}

		err = d.DB.DB().Ping()
		if err != nil {
			log.Printf("Could not Ping. Sleeping for 10s %s\n", err)
			time.Sleep(10 * time.Second)
			continue
		}

		if hasNodes := d.DB.HasTable(&models.NodeEntity{}); !hasNodes {
			if err = d.DB.CreateTable(&models.NodeEntity{}).Error; err != nil {
				log.Printf("Error creating Node table: %s\n", err)
			}
		}

		if hasPools := d.DB.HasTable(&models.PoolEntity{}); !hasPools {
			if err = d.DB.CreateTable(&models.PoolEntity{}).Error; err != nil {
				log.Printf("Error creating Pool table: %s\n", err)
			}
		}

		if hasSubnets := d.DB.HasTable(&models.SubnetEntity{}); !hasSubnets {
			if err = d.DB.CreateTable(&models.SubnetEntity{}).Error; err != nil {
				log.Printf("Error creating Pool table: %s\n", err)
			}
		}

		if err == nil {
			return nil
		}
	}
	return fmt.Errorf("Could not connect to database: %s\n", err)
}
