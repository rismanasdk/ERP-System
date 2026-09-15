package sales

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"time"

	"erp-system/backend/internal/audit"
	"erp-system/backend/internal/auth"
	"erp-system/backend/internal/branches"
	"erp-system/backend/internal/inventory"
	"erp-system/backend/internal/master/customers"
	"erp-system/backend/internal/master/products"
)

type inventoryRepository interface {
	GetByProductAndBranchForUpdate(ctx context.Context, tx *sql.Tx, productID, branchID int64) (*inventory.Inventory, error)
	UpdateQuantityWithTx(ctx context.Context, tx *sql.Tx, id, quantity int64) error
	CreateMovementWithTx(ctx context.Context, tx *sql.Tx, movement *inventory.StockMovement) (int64, error)
	EnsureInventoryWithTx(ctx context.Context, tx *sql.Tx, productID, branchID int64) (*inventory.Inventory, error)
}

type customerService interface {
	GetByID(ctx context.Context, id int64) (*customers.Customer, error)
}

type productService interface {
	GetByID(ctx context.Context, id int64) (*products.Product, error)
}

type branchService interface {
	EnsureUserHasAccess(ctx context.Context, userID, branchID int64, requireActive bool) error
	ListAccessibleBranches(ctx context.Context, filter branches.BranchFilter, userID int64) ([]branches.Branch, error)
}

type auditService interface {
	RecordWithTx(ctx context.Context, tx *sql.Tx, auditLog audit.AuditLog) (int64, error)
}

type repositoryInterface interface {
	BeginTx(ctx context.Context) (*sql.Tx, error)
	CreateSaleWithTx(ctx context.Context, tx *sql.Tx, sale *Sale) (int64, error)
	CreateSaleItemWithTx(ctx context.Context, tx *sql.Tx, item *SaleItem) (int64, error)
	GetSaleByID(ctx context.Context, id int64) (*Sale, error)
	GetSaleByIDForUpdate(ctx context.Context, tx *sql.Tx, id int64) (*Sale, error)
	GetSaleByNumber(ctx context.Context, number string) (*Sale, error)
	ListSales(ctx context.Context, filter SaleFilter) ([]Sale, error)
	ListSaleItemsBySaleID(ctx context.Context, saleID int64) ([]SaleItem, error)
	ListSaleItemsBySaleIDWithTx(ctx context.Context, tx *sql.Tx, saleID int64) ([]SaleItem, error)
	UpdateSaleStatusWithTx(ctx context.Context, tx *sql.Tx, id int64, status string) error
	CreateFulfillmentWithTx(ctx context.Context, tx *sql.Tx, fulfillment *SaleFulfillment) (int64, error)
	CreateFulfillmentItemWithTx(ctx context.Context, tx *sql.Tx, item *SaleFulfillmentItem) (int64, error)
	UpdateSaleItemFulfilledWithTx(ctx context.Context, tx *sql.Tx, itemID, fulfilled int64) error
	ListFulfillments(ctx context.Context, saleID int64) ([]SaleFulfillment, error)
	CreatePaymentWithTx(ctx context.Context, tx *sql.Tx, payment *SalesPayment) (int64, error)
	SumPaymentsWithTx(ctx context.Context, tx *sql.Tx, saleID int64) (float64, error)
	SumPayments(ctx context.Context, saleID int64) (float64, error)
	ListPayments(ctx context.Context, saleID int64) ([]SalesPayment, error)
}

type Service struct {
	repo          repositoryInterface
	inventoryRepo inventoryRepository
	productSvc    productService
	branchSvc     branchService
	authChecker   auth.PermissionChecker
	auditSvc      auditService
	customerSvc   customerService
}

const (
	SaleCreatePermission         = "sales.create"
	SaleReadPermission           = "sales.read"
	SaleCompletePermission       = "sales.complete"
	SaleConfirmPermission        = "sales.confirm"
	SaleFulfillPermission        = "sales.fulfill"
	SaleCancelPermission         = "sales.cancel"
	SaleStatusDraft              = "DRAFT"
	SaleStatusConfirmed          = "CONFIRMED"
	SaleStatusPartiallyFulfilled = "PARTIALLY_FULFILLED"
	SaleStatusFulfilled          = "FULFILLED"
	SaleStatusCompleted          = "COMPLETED"
	SaleStatusCancelled          = "CANCELLED"
	saleNumberRetryLimit         = 3
	saleNumberRandomPrefix       = 8
)

var (
	ErrAuthenticationRequired      = errors.New("authentication required")
	ErrForbidden                   = errors.New("permission denied")
	ErrSaleNotFound                = errors.New("sale not found")
	ErrProductNotFound             = errors.New("product not found")
	ErrProductInactive             = errors.New("product is inactive")
	ErrSaleHasNoItems              = errors.New("sale has no items")
	ErrInsufficientStock           = errors.New("insufficient stock")
	ErrInventoryNotFound           = errors.New("inventory not found")
	ErrInvalidSaleTransition       = errors.New("invalid sale status transition")
	ErrSaleAlreadyCompleted        = errors.New("sale already completed")
	ErrSaleAlreadyCancelled        = errors.New("sale already cancelled")
	ErrSaleNotConfirmed            = errors.New("sale must be confirmed before fulfillment")
	ErrFulfillmentExceedsRemaining = errors.New("fulfilled quantity exceeds remaining quantity")
	ErrCustomerNotFound            = errors.New("customer not found")
	ErrCustomerInactive            = errors.New("customer is inactive")
	ErrPaymentSaleNotPayable       = errors.New("sales order is not payable in its current status")
	ErrPaymentOverpayment          = errors.New("payment exceeds remaining amount")
	ErrInvalidPaymentMethod        = errors.New("invalid payment method")
	ErrInvalidPaymentAmount        = errors.New("payment amount must be greater than zero")
)

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

type CreateSaleItemInput struct {
	ProductID int64   `json:"product_id"`
	Quantity  int64   `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
}

type CreateSaleInput struct {
	CustomerID int64                 `json:"customer_id"`
	BranchID   int64                 `json:"branch_id"`
	Notes      *string               `json:"notes,omitempty"`
	Items      []CreateSaleItemInput `json:"items"`
}

type FulfillSaleItemInput struct {
	SaleItemID int64 `json:"sales_order_item_id"`
	Quantity   int64 `json:"quantity_fulfilled"`
}
type FulfillSaleInput struct {
	Notes *string                `json:"notes,omitempty"`
	Items []FulfillSaleItemInput `json:"items"`
}

func NewService(repo repositoryInterface, inventoryRepo inventoryRepository, productSvc productService, branchSvc branchService, authChecker auth.PermissionChecker, auditSvc auditService, customerSvcs ...customerService) *Service {
	service := &Service{
		repo:          repo,
		inventoryRepo: inventoryRepo,
		productSvc:    productSvc,
		branchSvc:     branchSvc,
		authChecker:   authChecker,
		auditSvc:      auditSvc,
	}
	if len(customerSvcs) > 0 {
		service.customerSvc = customerSvcs[0]
	}
	return service
}

func (s *Service) CreateSale(ctx context.Context, input CreateSaleInput) (saleID int64, err error) {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok || userID == 0 {
		return 0, ErrAuthenticationRequired
	}

	allowed, err := s.authChecker.HasPermission(ctx, userID, SaleCreatePermission)
	if err != nil {
		return 0, err
	}
	if !allowed {
		return 0, ErrForbidden
	}

	if len(input.Items) == 0 {
		return 0, &ValidationError{Field: "items", Message: "must contain at least one item"}
	}
	if s.customerSvc != nil {
		if input.CustomerID <= 0 {
			return 0, &ValidationError{Field: "customer_id", Message: "must be provided"}
		}
		customer, customerErr := s.customerSvc.GetByID(ctx, input.CustomerID)
		if customerErr != nil {
			if errors.Is(customerErr, customers.ErrCustomerNotFound) {
				return 0, ErrCustomerNotFound
			}
			return 0, customerErr
		}
		if !customer.IsActive {
			return 0, ErrCustomerInactive
		}
	}
	if err = s.branchSvc.EnsureUserHasAccess(ctx, userID, input.BranchID, true); err != nil {
		return 0, err
	}

	existingProducts := map[int64]struct{}{}
	items := make([]SaleItem, 0, len(input.Items))
	totalAmount := float64(0)
	for _, itemInput := range input.Items {
		if itemInput.ProductID == 0 {
			return 0, &ValidationError{Field: "product_id", Message: "must be provided"}
		}
		if _, found := existingProducts[itemInput.ProductID]; found {
			return 0, errors.New("duplicate product in sale items")
		}
		existingProducts[itemInput.ProductID] = struct{}{}
		if itemInput.Quantity <= 0 {
			return 0, &ValidationError{Field: "quantity", Message: "must be greater than zero"}
		}
		if itemInput.UnitPrice < 0 {
			return 0, &ValidationError{Field: "unit_price", Message: "must be greater than or equal to 0"}
		}

		product, err := s.productSvc.GetByID(ctx, itemInput.ProductID)
		if err != nil {
			if errors.Is(err, products.ErrProductNotFound) {
				return 0, ErrProductNotFound
			}
			return 0, err
		}
		if !product.IsActive {
			return 0, ErrProductInactive
		}

		subtotal := float64(itemInput.Quantity) * itemInput.UnitPrice
		totalAmount += subtotal
		items = append(items, SaleItem{
			ProductID: itemInput.ProductID,
			Quantity:  itemInput.Quantity,
			UnitPrice: itemInput.UnitPrice,
			Subtotal:  subtotal,
		})
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return 0, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	sale := &Sale{
		CustomerID:  input.CustomerID,
		BranchID:    input.BranchID,
		SaleNumber:  generateSaleNumber(),
		Status:      SaleStatusDraft,
		TotalAmount: totalAmount,
		Notes:       input.Notes,
		CreatedBy:   userID,
	}
	for attempt := 1; attempt < saleNumberRetryLimit; attempt++ {
		if _, err = s.repo.GetSaleByNumber(ctx, sale.SaleNumber); err == nil {
			sale.SaleNumber = generateSaleNumber()
			continue
		} else if !errors.Is(err, sql.ErrNoRows) {
			return 0, err
		}
		break
	}

	saleID, err = s.repo.CreateSaleWithTx(ctx, tx, sale)
	if err != nil {
		return 0, err
	}
	for i := range items {
		items[i].SaleID = saleID
		if _, err = s.repo.CreateSaleItemWithTx(ctx, tx, &items[i]); err != nil {
			return 0, err
		}
	}

	if s.auditSvc != nil {
		resourceID := fmt.Sprintf("%d", saleID)
		_, err = s.auditSvc.RecordWithTx(ctx, tx, audit.AuditLog{
			ActorUserID: actorUserIDFromContext(ctx),
			Action:      "sale.create",
			Resource:    "sale",
			ResourceID:  &resourceID,
			Metadata: map[string]any{
				"branch_id":    input.BranchID,
				"total_amount": totalAmount,
				"item_count":   len(items),
			},
		})
		if err != nil {
			return 0, err
		}
	}

	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return saleID, nil
}

func (s *Service) GetSale(ctx context.Context, id int64) (sale *Sale, items []SaleItem, err error) {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok || userID == 0 {
		return nil, nil, ErrAuthenticationRequired
	}
	allowed, err := s.authChecker.HasPermission(ctx, userID, SaleReadPermission)
	if err != nil {
		return nil, nil, err
	}
	if !allowed {
		return nil, nil, ErrForbidden
	}

	sale, err = s.repo.GetSaleByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, ErrSaleNotFound
		}
		return nil, nil, err
	}
	if err = s.branchSvc.EnsureUserHasAccess(ctx, userID, sale.BranchID, true); err != nil {
		return nil, nil, err
	}

	items, err = s.repo.ListSaleItemsBySaleID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	return sale, items, nil
}

func (s *Service) ConfirmSale(ctx context.Context, saleID int64) error {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok || userID == 0 {
		return ErrAuthenticationRequired
	}
	allowed, err := s.authChecker.HasPermission(ctx, userID, SaleConfirmPermission)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrForbidden
	}
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	sale, err := s.repo.GetSaleByIDForUpdate(ctx, tx, saleID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrSaleNotFound
		}
		return err
	}
	if err = s.branchSvc.EnsureUserHasAccess(ctx, userID, sale.BranchID, true); err != nil {
		return err
	}
	if sale.Status != SaleStatusDraft {
		return ErrInvalidSaleTransition
	}
	items, err := s.repo.ListSaleItemsBySaleIDWithTx(ctx, tx, saleID)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return ErrSaleHasNoItems
	}
	if err = s.repo.UpdateSaleStatusWithTx(ctx, tx, saleID, SaleStatusConfirmed); err != nil {
		return err
	}
	if s.auditSvc != nil {
		resourceID := fmt.Sprintf("%d", saleID)
		if _, err = s.auditSvc.RecordWithTx(ctx, tx, audit.AuditLog{ActorUserID: actorUserIDFromContext(ctx), Action: "sale.confirm", Resource: "sale", ResourceID: &resourceID}); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Service) FulfillSale(ctx context.Context, saleID int64, input FulfillSaleInput) (fulfillmentID int64, err error) {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok || userID == 0 {
		return 0, ErrAuthenticationRequired
	}
	allowed, err := s.authChecker.HasPermission(ctx, userID, SaleFulfillPermission)
	if err != nil {
		return 0, err
	}
	if !allowed {
		return 0, ErrForbidden
	}
	if len(input.Items) == 0 {
		return 0, &ValidationError{Field: "items", Message: "must contain at least one item"}
	}
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return 0, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	sale, err := s.repo.GetSaleByIDForUpdate(ctx, tx, saleID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrSaleNotFound
		}
		return 0, err
	}
	if err = s.branchSvc.EnsureUserHasAccess(ctx, userID, sale.BranchID, true); err != nil {
		return 0, err
	}
	if sale.Status != SaleStatusConfirmed && sale.Status != SaleStatusPartiallyFulfilled {
		return 0, ErrSaleNotConfirmed
	}
	items, err := s.repo.ListSaleItemsBySaleIDWithTx(ctx, tx, saleID)
	if err != nil {
		return 0, err
	}
	byID := map[int64]SaleItem{}
	for _, item := range items {
		byID[item.ID] = item
	}
	seen := map[int64]bool{}
	for _, inputItem := range input.Items {
		item, found := byID[inputItem.SaleItemID]
		if !found {
			return 0, ErrSaleNotFound
		}
		if inputItem.Quantity <= 0 {
			return 0, &ValidationError{Field: "quantity_fulfilled", Message: "must be greater than zero"}
		}
		if seen[inputItem.SaleItemID] || inputItem.Quantity > item.Quantity-item.FulfilledQuantity {
			return 0, ErrFulfillmentExceedsRemaining
		}
		seen[inputItem.SaleItemID] = true
	}
	fulfillment := &SaleFulfillment{SaleID: saleID, BranchID: sale.BranchID, FulfilledBy: userID, Notes: input.Notes}
	fulfillmentID, err = s.repo.CreateFulfillmentWithTx(ctx, tx, fulfillment)
	if err != nil {
		return 0, err
	}
	allFulfilled := true
	for _, item := range items {
		if item.FulfilledQuantity+quantityForSaleItem(item.ID, input.Items) < item.Quantity {
			allFulfilled = false
		}
	}
	for _, inputItem := range input.Items {
		item := byID[inputItem.SaleItemID]
		inv, invErr := s.inventoryRepo.GetByProductAndBranchForUpdate(ctx, tx, item.ProductID, sale.BranchID)
		if invErr != nil {
			if errors.Is(invErr, sql.ErrNoRows) {
				return 0, ErrInventoryNotFound
			}
			return 0, invErr
		}
		if inv.Quantity < inputItem.Quantity {
			return 0, ErrInsufficientStock
		}
		if err = s.inventoryRepo.UpdateQuantityWithTx(ctx, tx, inv.ID, inv.Quantity-inputItem.Quantity); err != nil {
			return 0, err
		}
		if _, err = s.repo.CreateFulfillmentItemWithTx(ctx, tx, &SaleFulfillmentItem{FulfillmentID: fulfillmentID, SalesOrderID: saleID, SaleItemID: item.ID, ProductID: item.ProductID, QuantityFulfilled: inputItem.Quantity}); err != nil {
			return 0, err
		}
		if err = s.repo.UpdateSaleItemFulfilledWithTx(ctx, tx, item.ID, item.FulfilledQuantity+inputItem.Quantity); err != nil {
			return 0, err
		}
		movement := &inventory.StockMovement{ProductID: item.ProductID, BranchID: sale.BranchID, MovementType: "SALES_OUT", QuantityDelta: -inputItem.Quantity, ReferenceType: inventory.PtrString("sales_fulfillment"), ReferenceID: &fulfillmentID, ActorUserID: actorUserIDFromContext(ctx)}
		if _, err = s.inventoryRepo.CreateMovementWithTx(ctx, tx, movement); err != nil {
			return 0, err
		}
	}
	status := SaleStatusPartiallyFulfilled
	if allFulfilled {
		status = SaleStatusFulfilled
	}
	if err = s.repo.UpdateSaleStatusWithTx(ctx, tx, saleID, status); err != nil {
		return 0, err
	}
	if s.auditSvc != nil {
		resourceID := fmt.Sprintf("%d", fulfillmentID)
		if _, err = s.auditSvc.RecordWithTx(ctx, tx, audit.AuditLog{ActorUserID: actorUserIDFromContext(ctx), Action: "sale.fulfill", Resource: "sales_fulfillment", ResourceID: &resourceID, Metadata: map[string]any{"sale_id": saleID, "branch_id": sale.BranchID}}); err != nil {
			return 0, err
		}
	}
	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return fulfillmentID, nil
}

func quantityForSaleItem(id int64, items []FulfillSaleItemInput) int64 {
	for _, item := range items {
		if item.SaleItemID == id {
			return item.Quantity
		}
	}
	return 0
}

func (s *Service) CreatePayment(ctx context.Context, saleID int64, input CreatePaymentInput) (paymentID int64, err error) {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok || userID == 0 {
		return 0, ErrAuthenticationRequired
	}
	allowed, err := s.authChecker.HasPermission(ctx, userID, "sales.payment.create")
	if err != nil {
		return 0, err
	}
	if !allowed {
		return 0, ErrForbidden
	}
	amountCents, ok := moneyCents(input.Amount)
	if !ok || amountCents <= 0 {
		return 0, ErrInvalidPaymentAmount
	}
	if input.PaymentMethod != "CASH" && input.PaymentMethod != "BANK_TRANSFER" && input.PaymentMethod != "OTHER" && input.PaymentMethod != "QRIS" {
		return 0, ErrInvalidPaymentMethod
	}
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return 0, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	sale, err := s.repo.GetSaleByIDForUpdate(ctx, tx, saleID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrSaleNotFound
		}
		return 0, err
	}
	if err = s.branchSvc.EnsureUserHasAccess(ctx, userID, sale.BranchID, true); err != nil {
		return 0, err
	}
	if sale.Status == SaleStatusDraft || sale.Status == SaleStatusCancelled {
		return 0, ErrPaymentSaleNotPayable
	}
	paid, err := s.repo.SumPaymentsWithTx(ctx, tx, saleID)
	if err != nil {
		return 0, err
	}
	paidCents, paidOK := moneyCents(paid)
	totalCents, ok := moneyCents(sale.TotalAmount)
	if !ok || !paidOK || paidCents+amountCents > totalCents {
		return 0, ErrPaymentOverpayment
	}
	payment := &SalesPayment{SalesOrderID: saleID, Amount: input.Amount, PaymentMethod: input.PaymentMethod, ReferenceNumber: input.ReferenceNumber, Notes: input.Notes, CreatedBy: userID}
	if input.PaidAt != nil {
		payment.PaidAt = *input.PaidAt
	}
	paymentID, err = s.repo.CreatePaymentWithTx(ctx, tx, payment)
	if err != nil {
		return 0, err
	}
	if s.auditSvc != nil {
		resourceID := fmt.Sprintf("%d", paymentID)
		_, err = s.auditSvc.RecordWithTx(ctx, tx, audit.AuditLog{ActorUserID: actorUserIDFromContext(ctx), Action: "sale.payment.create", Resource: "sales_payment", ResourceID: &resourceID, Metadata: map[string]any{"sale_id": saleID, "customer_id": sale.CustomerID, "branch_id": sale.BranchID, "amount": input.Amount, "payment_method": input.PaymentMethod}})
		if err != nil {
			return 0, err
		}
	}
	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return paymentID, nil
}

func (s *Service) ListPayments(ctx context.Context, saleID int64) ([]SalesPayment, error) {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok || userID == 0 {
		return nil, ErrAuthenticationRequired
	}
	allowed, err := s.authChecker.HasPermission(ctx, userID, "sales.payment.read")
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrForbidden
	}
	sale, err := s.repo.GetSaleByID(ctx, saleID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSaleNotFound
		}
		return nil, err
	}
	if err = s.branchSvc.EnsureUserHasAccess(ctx, userID, sale.BranchID, true); err != nil {
		return nil, err
	}
	return s.repo.ListPayments(ctx, saleID)
}

func (s *Service) GetPaymentSummary(ctx context.Context, saleID int64) (*PaymentSummary, error) {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok || userID == 0 {
		return nil, ErrAuthenticationRequired
	}
	allowed, err := s.authChecker.HasPermission(ctx, userID, "sales.payment.read")
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrForbidden
	}
	sale, err := s.repo.GetSaleByID(ctx, saleID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSaleNotFound
		}
		return nil, err
	}
	if err = s.branchSvc.EnsureUserHasAccess(ctx, userID, sale.BranchID, true); err != nil {
		return nil, err
	}
	paid, err := s.repo.SumPayments(ctx, saleID)
	if err != nil {
		return nil, err
	}
	return buildPaymentSummary(sale.TotalAmount, paid), nil
}

func buildPaymentSummary(total, paid float64) *PaymentSummary {
	totalCents, totalOK := moneyCents(total)
	paidCents, paidOK := moneyCents(paid)
	if !totalOK || !paidOK {
		return &PaymentSummary{OrderTotal: total, PaidAmount: paid, RemainingAmount: 0, PaymentStatus: "UNPAID"}
	}
	remaining := totalCents - paidCents
	if remaining < 0 {
		remaining = 0
	}
	status := "UNPAID"
	if paidCents > 0 && remaining > 0 {
		status = "PARTIALLY_PAID"
	}
	if remaining == 0 && totalCents > 0 {
		status = "PAID"
	}
	return &PaymentSummary{OrderTotal: float64(totalCents) / 100, PaidAmount: float64(paidCents) / 100, RemainingAmount: float64(remaining) / 100, PaymentStatus: status}
}

func moneyCents(value float64) (int64, bool) {
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > float64(math.MaxInt64)/100 {
		return 0, false
	}
	return int64(math.Round(value * 100)), true
}

func (s *Service) ListFulfillments(ctx context.Context, saleID int64) ([]SaleFulfillment, error) {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok || userID == 0 {
		return nil, ErrAuthenticationRequired
	}
	allowed, err := s.authChecker.HasPermission(ctx, userID, SaleReadPermission)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrForbidden
	}
	sale, err := s.repo.GetSaleByID(ctx, saleID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSaleNotFound
		}
		return nil, err
	}
	if err = s.branchSvc.EnsureUserHasAccess(ctx, userID, sale.BranchID, true); err != nil {
		return nil, err
	}
	return s.repo.ListFulfillments(ctx, saleID)
}

func (s *Service) ListSales(ctx context.Context, filter SaleFilter) (sales []Sale, err error) {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok || userID == 0 {
		return nil, ErrAuthenticationRequired
	}
	allowed, err := s.authChecker.HasPermission(ctx, userID, SaleReadPermission)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrForbidden
	}

	if filter.BranchID != nil {
		if err := s.branchSvc.EnsureUserHasAccess(ctx, userID, *filter.BranchID, true); err != nil {
			return nil, err
		}
		return s.repo.ListSales(ctx, filter)
	}

	branchesList, err := s.branchSvc.ListAccessibleBranches(ctx, branches.BranchFilter{}, userID)
	if err != nil {
		return nil, err
	}
	if len(branchesList) == 0 {
		return []Sale{}, nil
	}
	var allSales []Sale
	for _, branch := range branchesList {
		filterCopy := filter
		filterCopy.BranchID = &branch.ID
		branchSales, err := s.repo.ListSales(ctx, filterCopy)
		if err != nil {
			return nil, err
		}
		allSales = append(allSales, branchSales...)
	}
	return allSales, nil
}

func (s *Service) CompleteSale(ctx context.Context, saleID int64) (err error) {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok || userID == 0 {
		return ErrAuthenticationRequired
	}

	allowed, err := s.authChecker.HasPermission(ctx, userID, SaleCompletePermission)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrForbidden
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	sale, err := s.repo.GetSaleByIDForUpdate(ctx, tx, saleID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrSaleNotFound
		}
		return err
	}
	if err = s.branchSvc.EnsureUserHasAccess(ctx, userID, sale.BranchID, true); err != nil {
		return err
	}

	switch sale.Status {
	case SaleStatusDraft:
		// allowed
	case SaleStatusCompleted:
		return ErrSaleAlreadyCompleted
	case SaleStatusCancelled:
		return ErrInvalidSaleTransition
	default:
		return ErrInvalidSaleTransition
	}

	items, err := s.repo.ListSaleItemsBySaleIDWithTx(ctx, tx, saleID)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return ErrSaleHasNoItems
	}

	for _, item := range items {
		inventoryRow, err := s.inventoryRepo.GetByProductAndBranchForUpdate(ctx, tx, item.ProductID, sale.BranchID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrInventoryNotFound
			}
			return err
		}
		if inventoryRow.Quantity < item.Quantity {
			return ErrInsufficientStock
		}
		newQuantity := inventoryRow.Quantity - item.Quantity
		if err = s.inventoryRepo.UpdateQuantityWithTx(ctx, tx, inventoryRow.ID, newQuantity); err != nil {
			return err
		}
		movement := &inventory.StockMovement{
			ProductID:     item.ProductID,
			BranchID:      sale.BranchID,
			MovementType:  inventory.MovementTypeOUT,
			QuantityDelta: -item.Quantity,
			ReferenceType: inventory.PtrString("sale"),
			ReferenceID:   inventory.PtrInt64(saleID),
			ActorUserID:   actorUserIDFromContext(ctx),
			Metadata: map[string]any{
				"sale_id":    saleID,
				"branch_id":  sale.BranchID,
				"product_id": item.ProductID,
				"quantity":   item.Quantity,
			},
		}
		if _, err = s.inventoryRepo.CreateMovementWithTx(ctx, tx, movement); err != nil {
			return err
		}
	}

	if err = s.repo.UpdateSaleStatusWithTx(ctx, tx, saleID, SaleStatusCompleted); err != nil {
		return err
	}

	if s.auditSvc != nil {
		resourceID := fmt.Sprintf("%d", saleID)
		_, err = s.auditSvc.RecordWithTx(ctx, tx, audit.AuditLog{
			ActorUserID: actorUserIDFromContext(ctx),
			Action:      "sale.complete",
			Resource:    "sale",
			ResourceID:  &resourceID,
			Metadata: map[string]any{
				"sale_id":   saleID,
				"branch_id": sale.BranchID,
				"status":    SaleStatusCompleted,
			},
		})
		if err != nil {
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (s *Service) CancelSale(ctx context.Context, saleID int64) (err error) {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok || userID == 0 {
		return ErrAuthenticationRequired
	}

	allowed, err := s.authChecker.HasPermission(ctx, userID, SaleCancelPermission)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrForbidden
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	sale, err := s.repo.GetSaleByIDForUpdate(ctx, tx, saleID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrSaleNotFound
		}
		return err
	}
	if err = s.branchSvc.EnsureUserHasAccess(ctx, userID, sale.BranchID, true); err != nil {
		return err
	}

	switch sale.Status {
	case SaleStatusDraft:
		if _, err = tx.ExecContext(ctx, `
            UPDATE sales
            SET status = $1, updated_at = NOW()
            WHERE id = $2
        `, SaleStatusCancelled, saleID); err != nil {
			return err
		}
	case SaleStatusCompleted:
		items, err := s.repo.ListSaleItemsBySaleIDWithTx(ctx, tx, saleID)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			return ErrSaleHasNoItems
		}
		for _, item := range items {
			inventoryRow, err := s.inventoryRepo.GetByProductAndBranchForUpdate(ctx, tx, item.ProductID, sale.BranchID)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return ErrInventoryNotFound
				}
				return err
			}
			newQuantity := inventoryRow.Quantity + item.Quantity
			if err = s.inventoryRepo.UpdateQuantityWithTx(ctx, tx, inventoryRow.ID, newQuantity); err != nil {
				return err
			}
			movement := &inventory.StockMovement{
				ProductID:     item.ProductID,
				BranchID:      sale.BranchID,
				MovementType:  inventory.MovementTypeIN,
				QuantityDelta: item.Quantity,
				ReferenceType: inventory.PtrString("sale_cancel"),
				ReferenceID:   inventory.PtrInt64(saleID),
				ActorUserID:   actorUserIDFromContext(ctx),
				Metadata: map[string]any{
					"sale_id":    saleID,
					"branch_id":  sale.BranchID,
					"product_id": item.ProductID,
					"quantity":   item.Quantity,
				},
			}
			if _, err = s.inventoryRepo.CreateMovementWithTx(ctx, tx, movement); err != nil {
				return err
			}
		}
		if _, err = tx.ExecContext(ctx, `
            UPDATE sales
            SET status = $1, updated_at = NOW()
            WHERE id = $2
        `, SaleStatusCancelled, saleID); err != nil {
			return err
		}
	case SaleStatusCancelled:
		return ErrSaleAlreadyCancelled
	default:
		return ErrInvalidSaleTransition
	}

	if s.auditSvc != nil {
		resourceID := fmt.Sprintf("%d", saleID)
		_, err = s.auditSvc.RecordWithTx(ctx, tx, audit.AuditLog{
			ActorUserID: actorUserIDFromContext(ctx),
			Action:      "sale.cancel",
			Resource:    "sale",
			ResourceID:  &resourceID,
			Metadata: map[string]any{
				"sale_id":   saleID,
				"branch_id": sale.BranchID,
				"status":    SaleStatusCancelled,
			},
		})
		if err != nil {
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

func generateSaleNumber() string {
	buf := make([]byte, saleNumberRandomPrefix)
	_, err := rand.Read(buf)
	if err != nil {
		return fmt.Sprintf("SALE-%d", time.Now().UTC().UnixNano())
	}
	return fmt.Sprintf("SALE-%s-%s", time.Now().UTC().Format("20060102150405"), hex.EncodeToString(buf))
}

func actorUserIDFromContext(ctx context.Context) *int64 {
	if userID, ok := auth.UserIDFromContext(ctx); ok {
		return &userID
	}
	return nil
}
