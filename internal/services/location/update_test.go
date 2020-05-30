package location

import (
	"context"
	"github.com/Pallinder/go-randomdata"
	"github.com/gidyon/pandemic-api/pkg/api/location"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ = Describe("Updating user #update", func() {
	var (
		updateReq *location.UpdateUserRequest
		ctx       context.Context
	)

	BeforeEach(func() {
		updateReq = &location.UpdateUserRequest{
			PhoneNumber: randomdata.PhoneNumber(),
			User:        fakeUser(),
		}
		ctx = context.Background()
	})

	Describe("Updating user with malformed request", func() {
		It("should fail when the request is nil", func() {
			updateReq = nil
			updateRes, err := LocationAPI.UpdateUser(ctx, updateReq)
			Expect(err).Should(HaveOccurred())
			Expect(status.Code(err)).Should(Equal(codes.InvalidArgument))
			Expect(updateRes).Should(BeNil())
		})
		It("should fail when phone number is missing", func() {
			updateReq.PhoneNumber = ""
			updateRes, err := LocationAPI.UpdateUser(ctx, updateReq)
			Expect(err).Should(HaveOccurred())
			Expect(status.Code(err)).Should(Equal(codes.InvalidArgument))
			Expect(updateRes).Should(BeNil())
		})
		It("should fail when user is nil", func() {
			updateReq.User = nil
			updateRes, err := LocationAPI.UpdateUser(ctx, updateReq)
			Expect(err).Should(HaveOccurred())
			Expect(status.Code(err)).Should(Equal(codes.InvalidArgument))
			Expect(updateRes).Should(BeNil())
		})
	})

	When("Updating user status with well formed request", func() {
		var (
			userPhone string
			userPB    *location.User
		)

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
				userPB = addReq.User
			})
		})

		Describe("Updating their account", func() {
			It("should succeed", func() {
				updateReq.PhoneNumber = userPhone
				updateRes, err := LocationAPI.UpdateUser(ctx, updateReq)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(status.Code(err)).Should(Equal(codes.OK))
				Expect(updateRes).ShouldNot(BeNil())
				userPhone = updateReq.User.PhoneNumber
			})
		})

		Describe("Lets get the user", func() {
			It("should succeed", func() {
				getReq := &location.GetUserRequest{
					PhoneNumber: userPhone,
				}
				getRes, err := LocationAPI.GetUser(ctx, getReq)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(status.Code(err)).Should(Equal(codes.OK))
				Expect(getRes).ShouldNot(BeNil())

				// The new user should have updated
				Expect(getRes.DeviceToken).ShouldNot(Equal(userPB.DeviceToken))
				Expect(getRes.FullName).ShouldNot(Equal(userPB.FullName))
			})
		})
	})
})
