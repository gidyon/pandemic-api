package location

import (
	"context"
	"github.com/Pallinder/go-randomdata"
	"github.com/gidyon/pandemic-api/pkg/api/location"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"math/rand"
)

func fakeUser() *location.User {
	return &location.User{
		PhoneNumber: randomdata.PhoneNumber()[:15],
		FullName:    randomdata.FullName(randomdata.Female),
		County:      randomdata.State(randomdata.Small),
		Status:      location.Status(rand.Intn(len(location.Status_value))),
		DeviceToken: randomdata.MacAddress(),
	}
}

var _ = Describe("Adding a user into the database #add", func() {
	var (
		addReq *location.AddUserRequest
		ctx    context.Context
	)

	BeforeEach(func() {
		addReq = &location.AddUserRequest{
			User: fakeUser(),
		}
		ctx = context.Background()
	})

	When("Adding user with malformed request", func() {
		It("should fail if the request is nil", func() {
			addReq = nil
			addRes, err := LocationAPI.AddUser(ctx, addReq)
			Expect(err).Should(HaveOccurred())
			Expect(status.Code(err)).Should(Equal(codes.InvalidArgument))
			Expect(addRes).Should(BeNil())
		})
		It("should fail if user is nil", func() {
			addReq.User = nil
			addRes, err := LocationAPI.AddUser(ctx, addReq)
			Expect(err).Should(HaveOccurred())
			Expect(status.Code(err)).Should(Equal(codes.InvalidArgument))
			Expect(addRes).Should(BeNil())
		})
		It("should fail if phone number is missing ", func() {
			addReq.User.PhoneNumber = ""
			addRes, err := LocationAPI.AddUser(ctx, addReq)
			Expect(err).Should(HaveOccurred())
			Expect(status.Code(err)).Should(Equal(codes.InvalidArgument))
			Expect(addRes).Should(BeNil())
		})
		It("should fail if full name is missing ", func() {
			addReq.User.FullName = ""
			addRes, err := LocationAPI.AddUser(ctx, addReq)
			Expect(err).Should(HaveOccurred())
			Expect(status.Code(err)).Should(Equal(codes.InvalidArgument))
			Expect(addRes).Should(BeNil())
		})
		It("should fail if county is missing ", func() {
			addReq.User.County = ""
			addRes, err := LocationAPI.AddUser(ctx, addReq)
			Expect(err).Should(HaveOccurred())
			Expect(status.Code(err)).Should(Equal(codes.InvalidArgument))
			Expect(addRes).Should(BeNil())
		})
	})

	When("Adding user with well-formed request", func() {
		var userPhone string
		It("should succeed", func() {
			addRes, err := LocationAPI.AddUser(ctx, addReq)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(status.Code(err)).Should(Equal(codes.OK))
			Expect(addRes).ShouldNot(BeNil())
			userPhone = addReq.User.PhoneNumber
		})

		When("Adding a user who exists, it should succeed but do an update", func() {
			It("should succeed", func() {
				addReq.User.PhoneNumber = userPhone
				addRes, err := LocationAPI.AddUser(ctx, addReq)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(status.Code(err)).Should(Equal(codes.OK))
				Expect(addRes).ShouldNot(BeNil())
			})
		})
	})

})
