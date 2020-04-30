package location

import (
	"context"
	"github.com/Pallinder/go-randomdata"
	"github.com/gidyon/pandemic-api/pkg/api/location"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ = Describe("Getting user #get", func() {
	var (
		getReq *location.GetUserRequest
		ctx    context.Context
	)

	BeforeEach(func() {
		getReq = &location.GetUserRequest{
			PhoneNumber: randomdata.PhoneNumber(),
		}
		ctx = context.Background()
	})

	Describe("Getting user with malformed request", func() {
		It("should fail when the request is nil", func() {
			getReq = nil
			getRes, err := LocationAPI.GetUser(ctx, getReq)
			Expect(err).Should(HaveOccurred())
			Expect(status.Code(err)).Should(Equal(codes.InvalidArgument))
			Expect(getRes).Should(BeNil())
		})
		It("should fail when phone number is missing", func() {
			getReq.PhoneNumber = ""
			getRes, err := LocationAPI.GetUser(ctx, getReq)
			Expect(err).Should(HaveOccurred())
			Expect(status.Code(err)).Should(Equal(codes.InvalidArgument))
			Expect(getRes).Should(BeNil())
		})
		It("should fail when user does not exist", func() {
			getRes, err := LocationAPI.GetUser(ctx, getReq)
			Expect(err).Should(HaveOccurred())
			Expect(status.Code(err)).Should(Equal(codes.NotFound))
			Expect(getRes).Should(BeNil())
		})
	})

	When("Getting user with well-formed request", func() {
		var userPhone string
		Describe("Create user first", func() {
			It("should succeed", func() {
				addReq := &location.AddUserRequest{
					User: fakeUser(),
				}
				addRes, err := LocationAPI.AddUser(ctx, addReq)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(status.Code(err)).Should(Equal(codes.OK))
				Expect(addRes).ShouldNot(BeNil())
				userPhone = addReq.User.PhoneNumber
			})
		})

		Describe("Getting the user", func() {
			It("should succeed", func() {
				getReq.PhoneNumber = userPhone
				getRes, err := LocationAPI.GetUser(ctx, getReq)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(status.Code(err)).Should(Equal(codes.OK))
				Expect(getRes).ShouldNot(BeNil())
			})
		})
	})
})
