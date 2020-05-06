package messaging

import (
	"github.com/Pallinder/go-randomdata"
	"github.com/gidyon/pandemic-api/pkg/api/messaging"
)

var _ = Describe("Sending alerts for contact tracing £alerts", func() {
	var (
		alertServer messaging.Messaging_AlertContactsServer
	)

	_ = alertServer

	Describe("Testing alertContacts method", func() {
		It("should send alert to user successfully", func() {
			contactData := &messaging.ContactData{
				Count:        20,
				UserPhone:    randomPhone(),
				FullName:     randomdata.FullName(randomdata.Male),
				PatientPhone: randomPhone(),
				DeviceToken:  randomdata.MacAddress(),
			}

			MessagingServer.alertContact(contactData)
		})
	})

})
