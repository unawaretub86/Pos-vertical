package repository

import (
	"context"
	"time"

	"database/sql"
	"github.com/andreyo/pos/internal/model"
)

type KDSRepository struct {
	db *sql.DB
}

func NewKDSRepository(db *sql.DB) *KDSRepository {
	return &KDSRepository{db: db}
}

func (r *KDSRepository) GetActiveOrders(ctx context.Context) (model.KDSResponse, error) {
	q := `
		SELECT
			o.id,
			o.type,
			COALESCE(t.name, ''),
			o.turn_number,
			o.created_at,
			i.id,
			p.name,
			i.quantity,
			i.note,
			i.status,
			i.created_at
		FROM orders o
		JOIN order_items i ON i.order_id = o.id
		LEFT JOIN dining_tables t ON o.table_id = t.id
		JOIN products p ON i.product_id = p.id
		WHERE o.status IN ('open', 'closed')
		  AND i.status IN ('pending', 'preparing', 'ready')
		ORDER BY o.created_at ASC, i.created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return model.KDSResponse{}, err
	}
	defer rows.Close()

	orderMap := make(map[int64]*model.KDSOrder)
	var orderedIDs []int64

	for rows.Next() {
		var oID int64
		var oType, tName string
		var tNum *int
		var oCreated time.Time
		
		var iID int64
		var pName string
		var iQty float64
		var iNote, iStatus string
		var iCreated time.Time

		err := rows.Scan(
			&oID, &oType, &tName, &tNum, &oCreated,
			&iID, &pName, &iQty, &iNote, &iStatus, &iCreated,
		)
		if err != nil {
			return model.KDSResponse{}, err
		}

		if _, exists := orderMap[oID]; !exists {
			orderMap[oID] = &model.KDSOrder{
				OrderID:    oID,
				Type:       oType,
				TableName:  tName,
				TurnNumber: tNum,
				CreatedAt:  oCreated,
				Items:      make([]model.KDSItem, 0),
			}
			orderedIDs = append(orderedIDs, oID)
		}

		item := model.KDSItem{
			ID:          iID,
			ProductName: pName,
			Quantity:    iQty,
			Note:        iNote,
			Status:      iStatus,
			CreatedAt:   iCreated,
		}
		
		order := orderMap[oID]
		order.Items = append(order.Items, item)
	}

	res := model.KDSResponse{
		ServerTime: time.Now().UTC(),
		Orders:     make([]model.KDSOrder, 0, len(orderedIDs)),
	}

	for _, id := range orderedIDs {
		res.Orders = append(res.Orders, *orderMap[id])
	}

	return res, nil
}

