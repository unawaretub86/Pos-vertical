package handler

import (
	"encoding/json"
	"net/http"
	"testing"
)

type kdsResponseJSON struct {
	ServerTime string         `json:"server_time"`
	Orders     []kdsOrderJSON `json:"orders"`
}

type kdsOrderJSON struct {
	OrderID    int64         `json:"order_id"`
	Type       string        `json:"type"`
	TableName  string        `json:"table_name"`
	TurnNumber *int          `json:"turn_number,omitempty"`
	CreatedAt  string        `json:"created_at"`
	Items      []kdsItemJSON `json:"items"`
}

type kdsItemJSON struct {
	ID          int64   `json:"id"`
	ProductName string  `json:"product_name"`
	Quantity    float64 `json:"quantity"`
	Note        string  `json:"note"`
	Status      string  `json:"status"`
	CreatedAt   string  `json:"created_at"`
}

// T-1: AC-3 Pedido closed con ítem pending -> devuelto
func TestKDS_T1_ClosedOrderWithPendingItem(t *testing.T) {
	f := newOrderFixture(t)
	tableID := f.mkTable("Mesa KDS 1")
	
	// Create order manually
	f.exec(`INSERT INTO orders (id, type, table_id, status, user_id, created_at) VALUES (100, 'table', ?, 'closed', 1, '2026-01-01 10:00:00')`, tableID)
	// pending item
	f.exec(`INSERT INTO order_items (id, order_id, product_id, quantity, unit_price, status, created_at) VALUES (101, 100, ?, 1, 4500, 'pending', '2026-01-01 10:01:00')`, cafeID)

	w := f.do(http.MethodGet, "/kds/orders", "")
	if w.Code != http.StatusOK {
		t.Fatalf("GET /kds/orders: %d %s", w.Code, w.Body.String())
	}

	var wrap struct { Data kdsResponseJSON `json:"data"` }
	if err := json.Unmarshal(w.Body.Bytes(), &wrap); err != nil {
	
		t.Fatal(err)
	}
	
	if len(wrap.Data.Orders) != 1 {
		t.Fatalf("esperaba 1 pedido, vino %d", len(wrap.Data.Orders))
	}
	if wrap.Data.Orders[0].OrderID != 100 {
		t.Errorf("esperaba order_id 100")
	}
	if len(wrap.Data.Orders[0].Items) != 1 {
		t.Fatalf("esperaba 1 item, vinieron %d", len(wrap.Data.Orders[0].Items))
	}
	if wrap.Data.Orders[0].Items[0].ID != 101 {
		t.Errorf("esperaba item 101")
	}
}

// T-2: AC-2 Excluye void, ready, delivered (ready no se excluye según contrato AC-3, se excluye void y delivered)
// Espera: pending, preparing, ready -> se incluyen.
// En el test: 1 pending, 1 ready, 1 void. Respuesta = 2 items.
func TestKDS_T2_ExcludesVoidAndDelivered(t *testing.T) {
	f := newOrderFixture(t)
	tableID := f.mkTable("Mesa KDS 2")
	
	f.exec(`INSERT INTO orders (id, type, table_id, status, user_id, created_at) VALUES (102, 'table', ?, 'open', 1, '2026-01-01 10:00:00')`, tableID)
	f.exec(`INSERT INTO order_items (id, order_id, product_id, quantity, unit_price, status, created_at) VALUES (103, 102, ?, 1, 4500, 'pending', '2026-01-01 10:01:00')`, cafeID)
	f.exec(`INSERT INTO order_items (id, order_id, product_id, quantity, unit_price, status, created_at) VALUES (104, 102, ?, 1, 8000, 'ready', '2026-01-01 10:02:00')`, tortaID)
	f.exec(`INSERT INTO order_items (id, order_id, product_id, quantity, unit_price, status, created_at) VALUES (105, 102, ?, 1, 4500, 'void', '2026-01-01 10:03:00')`, cafeID)
	f.exec(`INSERT INTO order_items (id, order_id, product_id, quantity, unit_price, status, created_at) VALUES (106, 102, ?, 1, 4500, 'delivered', '2026-01-01 10:04:00')`, cafeID)

	w := f.do(http.MethodGet, "/kds/orders", "")
	var wrap struct { Data kdsResponseJSON `json:"data"` }
	json.Unmarshal(w.Body.Bytes(), &wrap)
	
	
	if len(wrap.Data.Orders) != 1 {
		t.Fatalf("esperaba 1 pedido, vino %d", len(wrap.Data.Orders))
	}
	if len(wrap.Data.Orders[0].Items) != 2 {
		t.Errorf("esperaba 2 items (pending, ready), vinieron %d", len(wrap.Data.Orders[0].Items))
	}
}

// T-3: AC-1, AC-2 Pedido cancelled excluye sus ítems
func TestKDS_T3_ExcludesCancelledOrder(t *testing.T) {
	f := newOrderFixture(t)
	tableID := f.mkTable("Mesa KDS 3")
	
	f.exec(`INSERT INTO orders (id, type, table_id, status, user_id) VALUES (107, 'table', ?, 'cancelled', 1)`, tableID)
	f.exec(`INSERT INTO order_items (id, order_id, product_id, quantity, unit_price, status) VALUES (108, 107, ?, 1, 4500, 'pending')`, cafeID)
	
	w := f.do(http.MethodGet, "/kds/orders", "")
	var wrap struct { Data kdsResponseJSON `json:"data"` }
	json.Unmarshal(w.Body.Bytes(), &wrap)
	
	
	if len(wrap.Data.Orders) != 0 {
		t.Errorf("esperaba 0 pedidos devueltos, vino %d", len(wrap.Data.Orders))
	}
}

// T-5: AC-1 Respuesta incluye server_time
func TestKDS_T5_IncludesServerTime(t *testing.T) {
	f := newOrderFixture(t)
	w := f.do(http.MethodGet, "/kds/orders", "")
	var wrap struct { Data kdsResponseJSON `json:"data"` }
	json.Unmarshal(w.Body.Bytes(), &wrap)
	
	
	if wrap.Data.ServerTime == "" {
		t.Errorf("server_time vacio")
	}
}


