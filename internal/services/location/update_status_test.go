package location

import (
	"context"
	"github.com/Pallinder/go-randomdata"
	"github.com/gidyon/pandemic-api/pkg/api/location"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ = Describe("Updating user status #update", func() {
	var (
		updateReq *location.UpdateUserStatusRequest
		ctx       context.Context
	)

	BeforeEach(func() {
		updateReq = &location.UpdateUserStatusRequest{
			PhoneNumber: randomdata.PhoneNumber(),
			Status:      location.Status_RECOVERED,
		}
		ctx = context.Background()
	})

	Describe("Updating user status with malformed request", func() {
		It("should fail when the request is nil", func() {
			updateReq = nil
			updateRes, err := LocationAPI.UpdateUserStatus(ctx, updateReq)
			Expect(err).Should(HaveOccurred())
			Expect(status.Code(err)).Should(Equal(codes.InvalidArgument))
			Expect(updateRes).Should(BeNil())
		})
		It("should fail when phone number is missing", func() {
			updateReq.PhoneNumber = ""
			updateRes, err := LocationAPI.UpdateUserStatus(ctx, updateReq)
			Expect(err).Should(HaveOccurred())
			Expect(status.Code(err)).Should(Equal(codes.InvalidArgument))
			Expect(updateRes).Should(BeNil())
		})
	})

	When("Updating user status with well formed request", func() {
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

		Describe("Updating their status ", func() {
			It("should succeed", func() {
				updateReq.PhoneNumber = userPhone
				updateRes, err := LocationAPI.UpdateUserStatus(ctx, updateReq)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(status.Code(err)).Should(Equal(codes.OK))
				Expect(updateRes).ShouldNot(BeNil())
			})
		})
	})
})
