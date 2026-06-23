package httphandler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ntt0601zcoder/go-clean-arch-template/internal/domain"
	"github.com/ntt0601zcoder/go-clean-arch-template/internal/infra/apperr"
	"github.com/ntt0601zcoder/go-clean-arch-template/internal/ports/inbound"
)

// timeFormat is the wire format for timestamps in responses.
const timeFormat = time.RFC3339

// AccountHandler is the Gin REST adapter over the account use case. It binds and
// validates wire input, delegates to the use case, and lets ginx.ErrorMiddleware
// render any returned error — handlers stay free of transport status codes.
type AccountHandler struct {
	uc inbound.AccountUseCase
}

// NewAccountHandler wires the handler to its use case.
func NewAccountHandler(uc inbound.AccountUseCase) *AccountHandler {
	return &AccountHandler{uc: uc}
}

// RegisterRoutes mounts the account CRUD endpoints onto rg.
func (h *AccountHandler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/accounts")
	g.POST("", h.Create)
	g.GET("/:id", h.Get)
	g.GET("", h.List)
	g.PATCH("/:id", h.Update)
	g.DELETE("/:id", h.Delete)
}

// Create creates a new account.
//
//	@Summary	Create an account
//	@Tags		accounts
//	@Accept		json
//	@Produce	json
//	@Param		body	body		CreateAccountRequest	true	"account to create"
//	@Success	201		{object}	AccountResponse
//	@Failure	400		{object}	httpx.ErrorResponse
//	@Failure	409		{object}	httpx.ErrorResponse
//	@Router		/accounts [post]
func (h *AccountHandler) Create(c *gin.Context) {
	var req CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperr.ErrInvalidRequest.Cause(err))
		return
	}
	account, err := h.uc.Create(c.Request.Context(), req.toDomain())
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, fromDomain(account))
}

// Get fetches one account by id.
//
//	@Summary	Get an account by id
//	@Tags		accounts
//	@Produce	json
//	@Param		id	path		string	true	"account id"
//	@Success	200	{object}	AccountResponse
//	@Failure	404	{object}	httpx.ErrorResponse
//	@Router		/accounts/{id} [get]
func (h *AccountHandler) Get(c *gin.Context) {
	account, err := h.uc.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, fromDomain(account))
}

// List returns a filtered, paginated page of accounts.
//
//	@Summary	List accounts
//	@Tags		accounts
//	@Produce	json
//	@Param		status	query		int	false	"filter by status (1=active,2=inactive,3=blocked,4=deleted)"
//	@Param		limit	query		int	false	"page size"
//	@Param		offset	query		int	false	"page offset"
//	@Success	200		{object}	ListAccountsResponse
//	@Failure	400		{object}	httpx.ErrorResponse
//	@Router		/accounts [get]
func (h *AccountHandler) List(c *gin.Context) {
	filter := domain.ListAccountFilter{
		Limit:  atoiDefault(c.Query("limit"), 0),
		Offset: atoiDefault(c.Query("offset"), 0),
	}
	if raw := c.Query("status"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil {
			c.Error(apperr.ErrInvalidRequest.Cause(err).WithMessage("invalid status"))
			return
		}
		st := domain.AccountStatus(v)
		filter.Status = &st
	}
	accounts, total, err := h.uc.List(c.Request.Context(), filter)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, ListAccountsResponse{
		Accounts: fromDomainList(accounts),
		Total:    total,
	})
}

// Update edits mutable profile fields of an account.
//
//	@Summary	Update an account
//	@Tags		accounts
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string					true	"account id"
//	@Param		body	body		UpdateAccountRequest	true	"fields to update"
//	@Success	200		{object}	AccountResponse
//	@Failure	400		{object}	httpx.ErrorResponse
//	@Failure	404		{object}	httpx.ErrorResponse
//	@Router		/accounts/{id} [patch]
func (h *AccountHandler) Update(c *gin.Context) {
	var req UpdateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperr.ErrInvalidRequest.Cause(err))
		return
	}
	account, err := h.uc.Update(c.Request.Context(), req.toDomain(c.Param("id")))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, fromDomain(account))
}

// Delete removes an account by id.
//
//	@Summary	Delete an account
//	@Tags		accounts
//	@Param		id	path	string	true	"account id"
//	@Success	204
//	@Failure	404	{object}	httpx.ErrorResponse
//	@Router		/accounts/{id} [delete]
func (h *AccountHandler) Delete(c *gin.Context) {
	if err := h.uc.Delete(c.Request.Context(), c.Param("id")); err != nil {
		c.Error(err)
		return
	}
	c.Status(http.StatusNoContent)
}

// atoiDefault parses s as an int, returning def when s is empty or invalid.
func atoiDefault(s string, def int) int {
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}
