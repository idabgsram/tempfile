package handlers

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/pandeptwidyaop/tempfile/internal/config"
	"github.com/pandeptwidyaop/tempfile/internal/models"
	"github.com/pandeptwidyaop/tempfile/internal/services"
	"github.com/pandeptwidyaop/tempfile/internal/utils"
)

// PasteHandler handles paste endpoints.
type PasteHandler struct {
	config          *config.Config
	pasteService    *services.PasteService
	templateService *services.TemplateService
}

// NewPasteHandler creates a PasteHandler.
func NewPasteHandler(cfg *config.Config, pasteSvc *services.PasteService, tmplSvc *services.TemplateService) *PasteHandler {
	return &PasteHandler{
		config:          cfg,
		pasteService:    pasteSvc,
		templateService: tmplSvc,
	}
}

// CreatePaste handles POST /paste.
func (h *PasteHandler) CreatePaste(c *fiber.Ctx) error {
	isWeb := strings.Contains(c.Get("Accept"), "text/html")

	content := c.FormValue("content")
	if content == "" && len(c.Body()) > 0 && strings.Contains(c.Get("Content-Type"), "text/plain") {
		content = string(c.Body())
	}

	created, err := h.pasteService.Create(content)
	if err != nil {
		if isWeb && h.templateService != nil {
			return h.templateService.RenderErrorPage(c, "Paste Failed", err.Error(), "Please check the content and try again.")
		}
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	if h.config.Debug {
		log.Printf("Paste created: id=%s size=%d", created.ID, created.Size)
	}

	if isWeb {
		return c.Redirect("/p/" + created.ID)
	}

	baseURL := utils.GetBaseURL(c, h.config.PublicURL)
	return c.JSON(models.PasteResponse{
		Message:   "Paste created",
		ID:        created.ID,
		URL:       baseURL + "/p/" + created.ID,
		RawURL:    baseURL + "/p/" + created.ID + "/raw",
		Size:      created.Size,
		SizeHuman: utils.FormatBytes(created.Size),
		ExpiresAt: created.ExpiresAt,
		ExpiresIn: fmt.Sprintf("%d hour(s)", h.config.FileExpiryHours),
	})
}

// ViewPaste handles GET /p/:id.
func (h *PasteHandler) ViewPaste(c *fiber.Ctx) error {
	id := c.Params("id")
	fetched, err := h.pasteService.Get(id)
	if err != nil {
		return h.notFound(c, err)
	}

	baseURL := utils.GetBaseURL(c, h.config.PublicURL)
	rawURL := baseURL + "/p/" + fetched.ID + "/raw"
	expiresIn := humanizeRemaining(fetched.ExpiresAt)
	expiresAt := fetched.ExpiresAt.Format(time.RFC3339)

	return h.templateService.RenderPastePage(c, baseURL, fetched.ID, fetched.Content, rawURL, expiresAt, expiresIn)
}

// ViewPasteRaw handles GET /p/:id/raw.
func (h *PasteHandler) ViewPasteRaw(c *fiber.Ctx) error {
	id := c.Params("id")
	fetched, err := h.pasteService.Get(id)
	if err != nil {
		c.Status(fiber.StatusNotFound)
		return c.SendString("paste not found")
	}
	c.Set("Content-Type", "text/plain; charset=utf-8")
	return c.SendString(fetched.Content)
}

func (h *PasteHandler) notFound(c *fiber.Ctx, cause error) error {
	if errors.Is(cause, services.ErrInvalidPasteID) {
		return h.templateService.RenderErrorPage(c, "Invalid Paste", "The paste id is not valid.", "")
	}
	return h.templateService.RenderErrorPage(c, "Paste Not Found", "This paste has expired or never existed.", "")
}

func humanizeRemaining(expiresAt time.Time) string {
	d := time.Until(expiresAt)
	if d <= 0 {
		return "expired"
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
}
