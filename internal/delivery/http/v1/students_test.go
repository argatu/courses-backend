package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/assert/v2"
	"github.com/golang/mock/gomock"
	"github.com/zhashkevych/courses-backend/internal/domain"
	"github.com/zhashkevych/courses-backend/internal/service"
	mock_service "github.com/zhashkevych/courses-backend/internal/service/mocks"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"net/http/httptest"
	"testing"
	"time"
)

const (
	paymentLink = "http://payment.link/"
)

func TestHandler_studentCreateOrder(t *testing.T) {
	type mockBehavior func(r *mock_service.MockOrders, studentId, offerId, promoId primitive.ObjectID)

	studentId := primitive.NewObjectID()
	offerId := primitive.NewObjectID()
	promoId := primitive.NewObjectID()

	tests := []struct {
		name         string
		body         string
		studentId    primitive.ObjectID
		offerId      primitive.ObjectID
		promoId      primitive.ObjectID
		mockBehavior mockBehavior
		statusCode   int
		responseBody string
	}{
		{
			name:      "ok",
			body:      fmt.Sprintf(`{"offerId": "%s"}`, offerId.Hex()),
			studentId: studentId,
			offerId:   offerId,
			mockBehavior: func(r *mock_service.MockOrders, studentId, offerId, promoId primitive.ObjectID) {
				r.EXPECT().Create(context.Background(), studentId, offerId, promoId).Return(paymentLink, nil)
			},
			statusCode:   200,
			responseBody: fmt.Sprintf(`{"paymentLink":"%s"}`, paymentLink),
		},
		{
			name:      "ok w/ promocode",
			body:      fmt.Sprintf(`{"offerId": "%s", "promoId": "%s"}`, offerId.Hex(), promoId.Hex()),
			studentId: studentId,
			offerId:   offerId,
			promoId:   promoId,
			mockBehavior: func(r *mock_service.MockOrders, studentId, offerId, promoId primitive.ObjectID) {
				r.EXPECT().Create(context.Background(), studentId, offerId, promoId).Return(paymentLink, nil)
			},
			statusCode:   200,
			responseBody: fmt.Sprintf(`{"paymentLink":"%s"}`, paymentLink),
		},
		{
			name:         "offerId missing",
			body:         fmt.Sprintf(`{"offerId": "", "promoId": "%s"}`, promoId.Hex()),
			mockBehavior: func(r *mock_service.MockOrders, studentId, offerId, promoId primitive.ObjectID) {},
			statusCode:   400,
			responseBody: `{"message":"invalid input body"}`,
		},
		{
			name:         "invalid offerId",
			body:         fmt.Sprintf(`{"offerId": "123", "promoId": "%s"}`, promoId.Hex()),
			mockBehavior: func(r *mock_service.MockOrders, studentId, offerId, promoId primitive.ObjectID) {},
			statusCode:   400,
			responseBody: `{"message":"invalid offer id"}`,
		},
		{
			name:         "invalid promoId",
			body:         fmt.Sprintf(`{"offerId": "%s", "promoId": "123"}`, offerId.Hex()),
			mockBehavior: func(r *mock_service.MockOrders, studentId, offerId, promoId primitive.ObjectID) {},
			statusCode:   400,
			responseBody: `{"message":"invalid promo id"}`,
		},
		{
			name:      "service error",
			body:      fmt.Sprintf(`{"offerId": "%s", "promoId": "%s"}`, offerId.Hex(), promoId.Hex()),
			studentId: studentId,
			offerId:   offerId,
			promoId:   promoId,
			mockBehavior: func(r *mock_service.MockOrders, studentId, offerId, promoId primitive.ObjectID) {
				r.EXPECT().Create(context.Background(), studentId, offerId, promoId).Return("", errors.New("failed to create order"))
			},
			statusCode:   500,
			responseBody: `{"message":"failed to create order"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Init Dependencies
			c := gomock.NewController(t)
			defer c.Finish()

			s := mock_service.NewMockOrders(c)
			tt.mockBehavior(s, tt.studentId, tt.offerId, tt.promoId)

			handler := Handler{ordersService: s}

			// Init Endpoint
			r := gin.New()
			r.POST("/order", func(c *gin.Context) {
				c.Set(studentCtx, tt.studentId.Hex())
			}, handler.studentCreateOrder)

			// Create Request
			w := httptest.NewRecorder()
			req := httptest.NewRequest("POST", "/order",
				bytes.NewBufferString(tt.body))

			// Make Request
			r.ServeHTTP(w, req)

			// Assert
			assert.Equal(t, w.Code, tt.statusCode)
			assert.Equal(t, w.Body.String(), tt.responseBody)
		})
	}
}

func TestHandler_studentGetPromocode(t *testing.T) {
	type mockBehavior func(r *mock_service.MockCourses, schoolId primitive.ObjectID, code string, promocode domain.Promocode)

	schoolId := primitive.NewObjectID()

	promocode := domain.Promocode{
		Code:               "GOGOGO25",
		DiscountPercentage: 25,
	}

	setResponseBody := func(promocode domain.Promocode) string {
		body, _ := json.Marshal(promocode)
		return string(body)
	}

	tests := []struct {
		name         string
		code         string
		schoolId     primitive.ObjectID
		promocode    domain.Promocode
		mockBehavior mockBehavior
		statusCode   int
		responseBody string
	}{
		{
			name:      "ok",
			code:      "GOGOGO25",
			schoolId:  schoolId,
			promocode: promocode,
			mockBehavior: func(r *mock_service.MockCourses, schoolId primitive.ObjectID, code string, promocode domain.Promocode) {
				r.EXPECT().GetPromocodeByCode(context.Background(), schoolId, code).Return(promocode, nil)
			},
			statusCode:   200,
			responseBody: setResponseBody(promocode),
		},
		{
			name:         "empty code",
			code:         "",
			schoolId:     schoolId,
			promocode:    promocode,
			mockBehavior: func(r *mock_service.MockCourses, schoolId primitive.ObjectID, code string, promocode domain.Promocode) {},
			statusCode:   404,
			responseBody: `404 page not found`,
		},
		{
			name:      "service error",
			code:      "GOGOGO25",
			schoolId:  schoolId,
			promocode: promocode,
			mockBehavior: func(r *mock_service.MockCourses, schoolId primitive.ObjectID, code string, promocode domain.Promocode) {
				r.EXPECT().GetPromocodeByCode(context.Background(), schoolId, code).Return(promocode, errors.New("failed to get promocode"))
			},
			statusCode:   500,
			responseBody: `{"message":"failed to get promocode"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Init Dependencies
			c := gomock.NewController(t)
			defer c.Finish()

			s := mock_service.NewMockCourses(c)
			tt.mockBehavior(s, tt.schoolId, tt.code, tt.promocode)

			handler := Handler{coursesService: s}

			// Init Endpoint
			r := gin.New()
			r.GET("/promocode/:code", func(c *gin.Context) {
				c.Set(schoolCtx, domain.School{
					ID: schoolId,
				})
			}, handler.studentGetPromocode)

			// Create Request
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", fmt.Sprintf("/promocode/%s", tt.code), nil)

			// Make Request
			r.ServeHTTP(w, req)

			// Assert
			assert.Equal(t, w.Code, tt.statusCode)
			assert.Equal(t, w.Body.String(), tt.responseBody)
		})
	}
}

func TestHandler_studentGetModuleOffers(t *testing.T) {
	type mockBehavior func(r *mock_service.MockCourses, schoolId, moduleId primitive.ObjectID, offers []domain.Offer)

	schoolId := primitive.NewObjectID()
	moduleId := primitive.NewObjectID()

	createdAt := time.Now()
	packageIds := []primitive.ObjectID{
		primitive.NewObjectID(), primitive.NewObjectID(),
	}

	tests := []struct {
		name         string
		moduleId     string
		schoolId     primitive.ObjectID
		offers       []domain.Offer
		mockBehavior mockBehavior
		statusCode   int
		responseBody string
	}{
		{
			name:     "ok",
			moduleId: moduleId.Hex(),
			schoolId: schoolId,
			offers: []domain.Offer{
				{
					Name:        "test offer",
					Description: "description",
					CreatedAt:   createdAt,
					SchoolID:    schoolId,
					PackageIDs:  packageIds,
					Price: domain.Price{
						Value:    6900,
						Currency: "USD",
					},
				},
			},
			mockBehavior: func(r *mock_service.MockCourses, schoolId, moduleId primitive.ObjectID, offers []domain.Offer) {
				r.EXPECT().GetModuleOffers(context.Background(), schoolId, moduleId).Return(offers, nil)
			},
			statusCode:   200,
			responseBody: fmt.Sprintf(`{"offers":[{"id":"000000000000000000000000","name":"test offer","description":"description","createdAt":"%s","price":{"value":6900,"currency":"USD"}}]}`, createdAt.Format(time.RFC3339)),
		},
		{
			name:         "invalid module id",
			moduleId:     "123",
			schoolId:     schoolId,
			mockBehavior: func(r *mock_service.MockCourses, schoolId, moduleId primitive.ObjectID, offers []domain.Offer) {},
			statusCode:   400,
			responseBody: `{"message":"invalid id param"}`,
		},
		{
			name:     "service error",
			moduleId: moduleId.Hex(),
			schoolId: schoolId,
			mockBehavior: func(r *mock_service.MockCourses, schoolId, moduleId primitive.ObjectID, offers []domain.Offer) {
				r.EXPECT().GetModuleOffers(context.Background(), schoolId, moduleId).Return(nil, errors.New("failed to get offers"))
			},
			statusCode:   500,
			responseBody: `{"message":"failed to get offers"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Init Dependencies
			c := gomock.NewController(t)
			defer c.Finish()

			s := mock_service.NewMockCourses(c)

			id, _ := primitive.ObjectIDFromHex(tt.moduleId)
			tt.mockBehavior(s, tt.schoolId, id, tt.offers)

			handler := Handler{coursesService: s}

			// Init Endpoint
			r := gin.New()
			r.GET("/modules/:id/offers", func(c *gin.Context) {
				c.Set(schoolCtx, domain.School{
					ID: schoolId,
				})
			}, handler.studentGetModuleOffers)

			// Create Request
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", fmt.Sprintf("/modules/%s/offers", tt.moduleId), nil)

			// Make Request
			r.ServeHTTP(w, req)

			// Assert
			assert.Equal(t, w.Code, tt.statusCode)
			assert.Equal(t, w.Body.String(), tt.responseBody)
		})
	}
}

func TestHandler_studentGetModuleLessons(t *testing.T) {
	type mockBehavior func(r *mock_service.MockStudents, schoolId, moduleId, studentId primitive.ObjectID, lessons []domain.Lesson)

	schoolId := primitive.NewObjectID()
	moduleId := primitive.NewObjectID()
	studentId := primitive.NewObjectID()

	tests := []struct {
		name         string
		moduleId     string
		schoolId     primitive.ObjectID
		studentId    primitive.ObjectID
		lessons      []domain.Lesson
		mockBehavior mockBehavior
		statusCode   int
		responseBody string
	}{
		{
			name:      "ok",
			moduleId:  moduleId.Hex(),
			schoolId:  schoolId,
			studentId: studentId,
			lessons: []domain.Lesson{
				{
					Name:      "test lesson",
					Position:  0,
					Published: true,
					Content:   "content",
				},
			},
			mockBehavior: func(r *mock_service.MockStudents, schoolId, studentId, moduleId primitive.ObjectID, lessons []domain.Lesson) {
				r.EXPECT().GetStudentModuleWithLessons(context.Background(), schoolId, studentId, moduleId).Return(lessons, nil)
			},
			statusCode:   200,
			responseBody: `{"lessons":[{"id":"000000000000000000000000","name":"test lesson","position":0,"published":true,"content":"content"}]}`,
		},
		{
			name:         "invalid module id",
			moduleId:     "123",
			schoolId:     schoolId,
			studentId:    studentId,
			mockBehavior: func(r *mock_service.MockStudents, schoolId, studentId, moduleId primitive.ObjectID, lessons []domain.Lesson) {},
			statusCode:   400,
			responseBody: `{"message":"invalid id param"}`,
		},
		{
			name:      "module is not available",
			moduleId:  moduleId.Hex(),
			schoolId:  schoolId,
			studentId: studentId,
			mockBehavior: func(r *mock_service.MockStudents, schoolId, studentId, moduleId primitive.ObjectID, lessons []domain.Lesson) {
				r.EXPECT().GetStudentModuleWithLessons(context.Background(), schoolId, studentId, moduleId).Return(lessons, service.ErrModuleIsNotAvailable)
			},
			statusCode:   403,
			responseBody: fmt.Sprintf(`{"message":"%s"}`, service.ErrModuleIsNotAvailable.Error()),
		},
		{
			name:      "service error",
			moduleId:  moduleId.Hex(),
			schoolId:  schoolId,
			studentId: studentId,
			mockBehavior: func(r *mock_service.MockStudents, schoolId, studentId, moduleId primitive.ObjectID, lessons []domain.Lesson) {
				r.EXPECT().GetStudentModuleWithLessons(context.Background(), schoolId, studentId, moduleId).Return(lessons, errors.New("failed to get module"))
			},
			statusCode:   500,
			responseBody: `{"message":"failed to get module"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Init Dependencies
			c := gomock.NewController(t)
			defer c.Finish()

			s := mock_service.NewMockStudents(c)

			id, _ := primitive.ObjectIDFromHex(tt.moduleId)
			tt.mockBehavior(s, tt.schoolId, tt.studentId, id, tt.lessons)

			handler := Handler{studentsService: s}

			// Init Endpoint
			r := gin.New()
			r.GET("/modules/:id/lessons", func(c *gin.Context) {
				c.Set(schoolCtx, domain.School{
					ID: schoolId,
				})
				c.Set(studentCtx, tt.studentId.Hex())
			}, handler.studentGetModuleLessons)

			// Create Request
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", fmt.Sprintf("/modules/%s/lessons", tt.moduleId), nil)

			// Make Request
			r.ServeHTTP(w, req)

			// Assert
			assert.Equal(t, w.Code, tt.statusCode)
			assert.Equal(t, w.Body.String(), tt.responseBody)
		})
	}
}

//func TestHandler_studentGetCourseById(t *testing.T) {
//	type mockBehavior func(r *mock_service.MockCourses, courseId primitive.ObjectID, modules []domain.Module)
//
//	now := time.Now()
//	course := domain.Course{
//		ID:          primitive.NewObjectID(),
//		Name:        "course 1",
//		Code:        "course-1",
//		Description: "description",
//		ImageURL:    "imageUrl",
//		CreatedAt:   now,
//	}
//	school := domain.School{
//		ID:      primitive.NewObjectID(),
//		Courses: []domain.Course{course},
//	}
//
//	tests := []struct {
//		name         string
//		courseId     string
//		school       domain.School
//		studentId    primitive.ObjectID
//		modules      []domain.Module
//		mockBehavior mockBehavior
//		statusCode   int
//		responseBody string
//	}{
//		{
//			name:     "ok",
//			courseId: course.ID.Hex(),
//			school:   school,
//			modules: []domain.Module{
//				{
//					Name:      "test module",
//					Position:  0,
//					Published: true,
//					Lessons: []domain.Lesson{
//						{
//							Name:      "test lesson",
//							Published: true,
//						},
//					},
//				},
//			},
//			mockBehavior: func(r *mock_service.MockCourses, courseId primitive.ObjectID, modules []domain.Module) {
//				r.EXPECT().GetCourseModules(context.Background(), course.ID).Return(modules, nil)
//			},
//			statusCode: 200,
//			responseBody: fmt.Sprintf(`{"course":{"id":"%s", "name":""},[{"id":"000000000000000000000000","name":"test lesson","position":0,"published":true,"content":"content"}]}`,
//				course.ID.Hex()),
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			// Init Dependencies
//			c := gomock.NewController(t)
//			defer c.Finish()
//
//			s := mock_service.NewMockStudents(c)
//
//			id, _ := primitive.ObjectIDFromHex(tt.moduleId)
//			tt.mockBehavior(s, tt.schoolId, tt.studentId, id, tt.lessons)
//
//			handler := Handler{studentsService: s}
//
//			// Init Endpoint
//			r := gin.New()
//			r.GET("/modules/:id/lessons", func(c *gin.Context) {
//				c.Set(schoolCtx, domain.School{
//					ID: schoolId,
//				})
//				c.Set(studentCtx, tt.studentId.Hex())
//			}, handler.studentGetModuleLessons)
//
//			// Create Request
//			w := httptest.NewRecorder()
//			req := httptest.NewRequest("GET", fmt.Sprintf("/modules/%s/lessons", tt.moduleId), nil)
//
//			// Make Request
//			r.ServeHTTP(w, req)
//
//			// Assert
//			assert.Equal(t, w.Code, tt.statusCode)
//			assert.Equal(t, w.Body.String(), tt.responseBody)
//		})
//	}
//}
