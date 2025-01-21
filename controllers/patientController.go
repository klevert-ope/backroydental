package controllers

import (
	"RoyDental/handlers"

	"github.com/gin-gonic/gin"
)

func SetupPatientRoutes(router *gin.Engine, patientHandler *handlers.PatientHandler, doctorHandler *handlers.DoctorHandler, insuranceCompanyHandler *handlers.InsuranceCompanyHandler, emergencyContactHandler *handlers.EmergencyContactHandler, examinationHandler *handlers.ExaminationHandler, billingHandler *handlers.BillingHandler, treatmentPlanHandler *handlers.TreatmentPlanHandler, appointmentHandler *handlers.AppointmentHandler) {
	// Define the routes directly on the router
	router.POST("/doctors", doctorHandler.CreateDoctor)
	router.GET("/doctors/:id", doctorHandler.GetDoctorByID)
	router.PUT("/doctors/:id", doctorHandler.UpdateDoctor)
	router.DELETE("/doctors/:id", doctorHandler.DeleteDoctor)
	router.GET("/doctors", doctorHandler.GetAllDoctors)

	router.POST("/patients", patientHandler.CreatePatient)
	router.GET("/patients/:patient_id", patientHandler.GetPatientByID)
	router.PUT("/patients/:patient_id", patientHandler.UpdatePatient)
	router.DELETE("/patients/:patient_id", patientHandler.DeletePatient)
	router.DELETE("/patients/:patient_id/related", patientHandler.DeletePatientAndRelated)
	router.GET("/patients", patientHandler.GetAllPatients)

	router.POST("/insurance_companies", insuranceCompanyHandler.CreateInsuranceCompany)
	router.GET("/insurance_companies/:id", insuranceCompanyHandler.GetInsuranceCompanyByID)
	router.PUT("/insurance_companies/:id", insuranceCompanyHandler.UpdateInsuranceCompany)
	router.DELETE("/insurance_companies/:id", insuranceCompanyHandler.DeleteInsuranceCompany)
	router.GET("/insurance_companies", insuranceCompanyHandler.GetAllInsuranceCompanies)

	router.POST("/patients/:patient_id/emergency_contacts", emergencyContactHandler.CreateEmergencyContact)
	router.GET("/patients/:patient_id/emergency_contacts", emergencyContactHandler.GetAllEmergencyContacts)
	router.GET("/patients/:patient_id/emergency_contacts/:emergency_contact_id", emergencyContactHandler.GetEmergencyContactByID)
	router.PUT("/patients/:patient_id/emergency_contacts/:emergency_contact_id", emergencyContactHandler.UpdateEmergencyContact)
	router.DELETE("/patients/:patient_id/emergency_contacts/:emergency_contact_id", emergencyContactHandler.DeleteEmergencyContact)

	router.POST("/patients/:patient_id/examinations", examinationHandler.CreateExamination)
	router.GET("/patients/:patient_id/examinations", examinationHandler.GetAllExaminations)
	router.GET("/patients/:patient_id/examinations/:examination_id", examinationHandler.GetExaminationByID)
	router.PUT("/patients/:patient_id/examinations/:examination_id", examinationHandler.UpdateExamination)
	router.DELETE("/patients/:patient_id/examinations/:examination_id", examinationHandler.DeleteExamination)

	router.POST("/patients/:patient_id/treatment_plans", treatmentPlanHandler.CreateTreatmentPlan)
	router.GET("/patients/:patient_id/treatment_plans", treatmentPlanHandler.GetAllTreatmentPlans)
	router.GET("/patients/:patient_id/treatment_plans/:treatment_plan_id", treatmentPlanHandler.GetTreatmentPlanByID)
	router.PUT("/patients/:patient_id/treatment_plans/:treatment_plan_id", treatmentPlanHandler.UpdateTreatmentPlan)
	router.DELETE("/patients/:patient_id/treatment_plans/:treatment_plan_id", treatmentPlanHandler.DeleteTreatmentPlan)

	router.POST("/billings", billingHandler.CreateBilling)
	router.GET("/billings/:id", billingHandler.GetBillingByID)
	router.PUT("/billings/:id", billingHandler.UpdateBilling)
	router.DELETE("/billings/:id", billingHandler.DeleteBilling)
	router.GET("/billings", billingHandler.GetAllBillings)

	router.POST("/patients/:patient_id/appointments", appointmentHandler.CreateAppointment)
	router.GET("/patients/:patient_id/appointments", appointmentHandler.GetAllAppointments)
	router.GET("/patients/:patient_id/appointments/:appointment_id", appointmentHandler.GetAppointmentByID)
	router.PUT("/patients/:patient_id/appointments/:appointment_id", appointmentHandler.UpdateAppointment)
	router.DELETE("/patients/:patient_id/appointments/:appointment_id", appointmentHandler.DeleteAppointment)
}
