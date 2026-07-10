package campuslogin

import (
	"diaxel/internal/config"
	"diaxel/internal/constants"
	"diaxel/internal/grpc/db"
	"diaxel/internal/modules/llm"
	twilio "diaxel/internal/modules/twilio"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type CampusLoginHandler struct {
	cfg    *config.Settings
	db     *db.Client
	twilio *twilio.Client
	LLM    *llm.Client
}

func NewCampusLoginHandler(cfg *config.Settings, db *db.Client, twilioClient *twilio.Client, llmClient *llm.Client) *CampusLoginHandler {
	return &CampusLoginHandler{
		cfg:    cfg,
		db:     db,
		twilio: twilioClient,
		LLM:    llmClient,
	}
}

type CampusWebhookRequest struct {
	ContactID        string  `form:"ContactID" json:"ContactID"`
	CampusID         string  `form:"CampusID" json:"CampusID"`
	FirstName        string  `form:"FirstName" json:"FirstName"`
	LastName         string  `form:"Lastname" json:"Lastname"`
	AlternatePhone   string  `form:"alternatephone" json:"alternatephone"`
	Telephone        string  `form:"Telephone" json:"Telephone"`
	Email            string  `form:"Email" json:"Email"`
	StudentNumber    string  `form:"StudentNumber" json:"StudentNumber"`
	ID               string  `form:"ID" json:"ID"`
	ProgramID        string  `form:"ProgramID" json:"ProgramID"`
	AttributeID_2577 *string `form:"AttributeID_2577" json:"AttributeID_2577"`
}

func (h *CampusLoginHandler) HandleTriggerTwilio(c *gin.Context) {
	assistantID := c.Param("assistant_id")
	if assistantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "assistant_id parameter is required"})
		return
	}

	// --- ДОБАВЛЕННЫЙ БЛОК ДЛЯ ЛОГИРОВАНИЯ ВСЕХ ПОЛЕЙ ---
	if err := c.Request.ParseForm(); err == nil {
		log.Printf("[CampusLogin Debug] === НАЧАЛО ПРИНЯТЫХ ДАННЫХ ===")

		// Выводим всё, что пришло в URL (Query параметры)
		if len(c.Request.URL.Query()) > 0 {
			log.Printf("[CampusLogin Debug] URL Query параметры:")
			for key, values := range c.Request.URL.Query() {
				log.Printf("  %s: %s", key, strings.Join(values, ", "))
			}
		}

		if len(c.Request.PostForm) > 0 {
			log.Printf("[CampusLogin Debug] Form/Post параметры:")
			for key, values := range c.Request.PostForm {
				log.Printf("  %s: %s", key, strings.Join(values, ", "))
			}
		}
		log.Printf("[CampusLogin Debug] === КОНЕЦ ПРИНЯТЫХ ДАННЫХ ===")
	} else {
		log.Printf("[CampusLogin Debug] Не удалось распарсить форму для логирования: %v", err)
	}
	// ---------------------------------------------------

	//if assistantID == "test" {
	//	client := campusloginModule.NewClient(h.cfg.CampusLoginAPI)
	//	err := client.SendAppointment(c.Request.Context(), "2026-05-25T11:30:00", "2026-05-25T12:30:00", 5972449, 1, "Test appointment from API endpoint")
	//	if err != nil {
	//		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Test failed: %v", err)})
	//		return
	//	}
	//	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Test appointment successfully sent to CampusLogin"})
	//	return
	//}

	var req CampusWebhookRequest
	if err := c.ShouldBind(&req); err != nil {
		log.Printf("[CampusLogin Trigger] Binding error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data: " + err.Error()})
		return
	}

	toPhone := req.AlternatePhone
	if toPhone == "" {
		toPhone = req.Telephone
	}
	if toPhone == "" {
		log.Printf("[CampusLogin Trigger] Alternate phone and telephone numbers are missing")
		c.JSON(http.StatusBadRequest, gin.H{"error": "alternatephone or telephone is required"})
		return
	}

	contactIDInt := 0
	if req.ContactID != "" {
		parsed, err := strconv.Atoi(req.ContactID)
		if err == nil {
			contactIDInt = parsed
		} else {
			log.Printf("[CampusLogin Trigger] Failed to parse ContactID '%s': %v", req.ContactID, err)
		}
	}

	var digitsOnly string
	for _, r := range toPhone {
		if r >= '0' && r <= '9' {
			digitsOnly += string(r)
		}
	}

	if len(digitsOnly) == 10 {
		toPhone = "+1" + digitsOnly
	} else if len(digitsOnly) == 11 && strings.HasPrefix(digitsOnly, "1") {
		toPhone = "+" + digitsOnly
	} else {
		// Fallback
		if !strings.HasPrefix(toPhone, "+") {
			toPhone = "+" + toPhone
		}
	}

	isBlocked, err := h.db.IsCustomerBlocked(toPhone)
	if err == nil && isBlocked {
		log.Printf("[CampusLogin Trigger] Ignoring trigger for blocked customer %s", toPhone)
		c.JSON(http.StatusOK, gin.H{"status": "ignored", "message": "customer is blocked"})
		return
	}

	programIDInt := 0
	if req.ProgramID != "" {
		parsed, err := strconv.Atoi(req.ProgramID)
		if err == nil {
			programIDInt = parsed
		} else {
			log.Printf("[CampusLogin Trigger] Failed to parse ProgramID '%s': %v", req.ProgramID, err)
		}
	}

	if contactIDInt > 0 {
		err := h.db.UpsertCampuslogin(toPhone, contactIDInt, programIDInt, false, false, req.FirstName, req.Email)
		if err != nil {
			log.Printf("[CampusLogin Trigger] Failed to upsert Campuslogin for %s: %v", toPhone, err)
		}
	}

	//if toPhone != "+12048176146" && toPhone != "+12045909711" && toPhone != "+12045589015" {
	//	log.Printf("[CampusLogin Trigger] Ignoring message from unknown number %s", toPhone)
	//	c.JSON(http.StatusOK, gin.H{"status": "ignored", "message": "phone number not in allowed list"})
	//	return
	//}

	existingChat, checkErr := h.db.GetLatestChatByCustomer(assistantID, toPhone)
	if checkErr != nil {
		log.Printf("[CampusLogin Trigger] Error checking existing chat for %s: %v", toPhone, checkErr)
	} else if existingChat != nil && existingChat.Id != "" {
		log.Printf("[CampusLogin Trigger] Existing chat found (ID: %s) for %s. Deleting chat and messages.", existingChat.Id, toPhone)
		delErr := h.db.DeleteChatAndMessages(existingChat.Id)
		if delErr != nil {
			log.Printf("[CampusLogin Trigger] Failed to delete existing chat %s: %v", existingChat.Id, delErr)
		}
	}

	_, err = h.db.GetAssistant(assistantID)
	if err != nil {
		log.Printf("Error getting assistant %s: %v", assistantID, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "assistant not found"})
		return
	}

	twilioConfig, err := h.db.GetTwilioConfig(assistantID)
	if err != nil {
		log.Printf("Error getting twilio config for assistant %s: %v", assistantID, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "twilio configuration not found for this assistant"})
		return
	}

	programName := req.ProgramID
	if name, ok := constants.ProgramIDToName[req.ProgramID]; ok {
		programName = name
	}

	isInternational := "no"
	if req.AttributeID_2577 != nil {
		isInternational = "yes"
	}

	systemPrompt := fmt.Sprintf(
		"This is a new lead. Name: %s, program: %s, International: %s. Greet them by name and mention the program they chose.",
		req.FirstName,
		programName,
		isInternational,
	)

	log.Printf("[CampusLogin Trigger] Generated System Prompt: %s", systemPrompt)

	answer, err := h.LLM.Conversation(c, toPhone, assistantID, "", llm.WithSystemMessage(systemPrompt))
	if err != nil {
		log.Printf("LLM Conversation error for assistant %s: %v", assistantID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	err = h.twilio.SendMessage(c,
		twilioConfig.AccountSid,
		twilioConfig.AuthToken,
		twilioConfig.TwilioNumber,
		toPhone,
		answer,
	)
	if err != nil {
		log.Printf("Error sending Twilio message to %s: %v", toPhone, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to send message: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "Triggered successfully",
		"to":      toPhone,
		"answer":  answer,
	})
}

func (h *CampusLoginHandler) HandleTriggerTwilioReinquiry(c *gin.Context) {
	_ = c.Request.ParseMultipartForm(32 << 20)
	_ = c.Request.ParseForm()
	body, _ := c.GetRawData()

	log.Printf("====== ECHO REQUEST ======\n")
	log.Printf("Method: %s\n", c.Request.Method)
	log.Printf("URL: %s\n", c.Request.URL.String())
	log.Printf("Host: %s\n", c.Request.Host)
	log.Printf("RemoteAddr: %s\n", c.Request.RemoteAddr)
	log.Printf("ClientIP: %s\n", c.ClientIP())
	log.Printf("Headers: %+v\n", c.Request.Header)
	log.Printf("Query Params: %+v\n", c.Request.URL.Query())
	log.Printf("Form: %+v\n", c.Request.Form)
	log.Printf("PostForm: %+v\n", c.Request.PostForm)
	log.Printf("MultipartForm: %+v\n", c.Request.MultipartForm)
	log.Printf("ContentLength: %d\n", c.Request.ContentLength)
	log.Printf("Body: %s\n", string(body))
	log.Printf("==========================\n")

	c.Status(200)
}
