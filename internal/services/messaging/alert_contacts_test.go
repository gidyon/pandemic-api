package messaging

import (
	"github.com/Pallinder/go-randomdata"
	"github.com/gidyon/pandemic-api/pkg/api/location"
)

var _ = Describe("Sending alerts for contact tracing £alerts", func() {
	var (
		alertServer location.Messaging_AlertContactsServer
	)

	_ = alertServer

	Describe("Testing alertContacts method", func() {
		It("should send alert to user successfully", func() {
			contactData := &location.ContactData{
				Count:        20,
				UserPhone:    randomdata.PhoneNumber(),
				FullName:     randomdata.FullName(randomdata.Male),
				PatientPhone: randomdata.PhoneNumber(),
				DeviceToken:  randomdata.MacAddress(),
			}

			MessagingServer.alertContact(contactData)
		})
	})

})
