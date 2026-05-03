package example

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/MrMiaoMIMI/goshared/db/dbhelper"
	"github.com/MrMiaoMIMI/goshared/db/dbspi"
)

type Shop struct {
	ID         int64  `gorm:"primaryKey"`
	Name       string `gorm:"column:name"`
	OwnerEmail string `gorm:"column:owner_email"`
}

func (*Shop) TableName() string   { return "shop_tab" }
func (*Shop) IdFieldName() string { return dbspi.DefaultIdFieldName }

var ShopFields = struct {
	Name       dbspi.Field[string]
	OwnerEmail dbspi.Field[string]
}{
	Name:       dbhelper.NewField[string]("name"),
	OwnerEmail: dbhelper.NewField[string]("owner_email"),
}

func Test_Transaction_MultiTable_Commit(t *testing.T) {
	ctx := context.Background()
	mgr := testManager(testDatabaseName)
	ensureShopTable(t, ctx, mgr)

	userExec := dbhelper.NewExecutor(&User{}, dbhelper.WithManager(mgr))
	shopExec := dbhelper.NewExecutor(&Shop{}, dbhelper.WithManager(mgr))

	suffix := time.Now().UnixNano()
	email := fmt.Sprintf("tx_commit_%d@example.com", suffix)
	shopName := fmt.Sprintf("tx_commit_shop_%d", suffix)
	cleanupTransactionRows(t, ctx, userExec, shopExec, email, shopName)
	t.Cleanup(func() {
		cleanupTransactionRows(t, ctx, userExec, shopExec, email, shopName)
	})

	err := dbhelper.Transaction(ctx, func(tx *dbhelper.Tx) error {
		txUserExec := dbhelper.NewExecutor(&User{}, dbhelper.WithTx(tx))
		txShopExec := dbhelper.NewExecutor(&Shop{}, dbhelper.WithTx(tx))

		if err := txUserExec.Create(ctx, &User{Name: "Tx Commit User", Email: email, Age: 20, Status: "active"}); err != nil {
			return err
		}
		return txShopExec.Create(ctx, &Shop{Name: shopName, OwnerEmail: email})
	}, dbhelper.WithManager(mgr))
	requireNoError(t, err)

	userExists, _, err := userExec.Exists(ctx, dbhelper.Q(NewUserFieldManager().Email.Eq(&email)))
	requireNoError(t, err)
	if !userExists {
		t.Fatal("expected committed user row")
	}
	shopExists, _, err := shopExec.Exists(ctx, dbhelper.Q(ShopFields.Name.Eq(&shopName)))
	requireNoError(t, err)
	if !shopExists {
		t.Fatal("expected committed shop row")
	}
}

func Test_Transaction_MultiTable_Rollback(t *testing.T) {
	ctx := context.Background()
	mgr := testManager(testDatabaseName)
	ensureShopTable(t, ctx, mgr)

	userExec := dbhelper.NewExecutor(&User{}, dbhelper.WithManager(mgr))
	shopExec := dbhelper.NewExecutor(&Shop{}, dbhelper.WithManager(mgr))

	suffix := time.Now().UnixNano()
	email := fmt.Sprintf("tx_rollback_%d@example.com", suffix)
	shopName := fmt.Sprintf("tx_rollback_shop_%d", suffix)
	cleanupTransactionRows(t, ctx, userExec, shopExec, email, shopName)

	rollbackErr := errors.New("rollback transaction")
	err := dbhelper.Transaction(ctx, func(tx *dbhelper.Tx) error {
		txUserExec := dbhelper.NewExecutor(&User{}, dbhelper.WithTx(tx))
		txShopExec := dbhelper.NewExecutor(&Shop{}, dbhelper.WithTx(tx))

		if err := txUserExec.Create(ctx, &User{Name: "Tx Rollback User", Email: email, Age: 20, Status: "active"}); err != nil {
			return err
		}
		if err := txShopExec.Create(ctx, &Shop{Name: shopName, OwnerEmail: email}); err != nil {
			return err
		}
		return rollbackErr
	}, dbhelper.WithManager(mgr))
	if !errors.Is(err, rollbackErr) {
		t.Fatalf("transaction error = %v, want rollbackErr", err)
	}

	userExists, _, err := userExec.Exists(ctx, dbhelper.Q(NewUserFieldManager().Email.Eq(&email)))
	requireNoError(t, err)
	if userExists {
		t.Fatal("rollback should remove user row")
	}
	shopExists, _, err := shopExec.Exists(ctx, dbhelper.Q(ShopFields.Name.Eq(&shopName)))
	requireNoError(t, err)
	if shopExists {
		t.Fatal("rollback should remove shop row")
	}
}

func Test_Transaction_ShardedSameDb(t *testing.T) {
	ctx := context.Background()
	mgr := dbhelper.NewManager(dbspi.DatabaseConfig{
		DatabaseGroups: map[string]dbspi.DatabaseGroupConfig{
			dbspi.DefaultDatabaseGroupKey: defaultTestDatabaseGroupConfig(),
			"order_dbs":                   orderShopTableEntry(10),
		},
	})
	orderExec := dbhelper.NewExecutor(&Order{}, dbhelper.WithManager(mgr))

	amount := time.Now().UnixNano()
	shopID1 := int64(12345)
	shopID2 := int64(12346)
	cleanupOrderRows(t, ctx, orderExec, shopID1, amount)
	cleanupOrderRows(t, ctx, orderExec, shopID2, amount)
	t.Cleanup(func() {
		cleanupOrderRows(t, ctx, orderExec, shopID1, amount)
		cleanupOrderRows(t, ctx, orderExec, shopID2, amount)
	})

	err := dbhelper.Transaction(ctx, func(tx *dbhelper.Tx) error {
		txOrderExec := dbhelper.NewExecutor(&Order{}, dbhelper.WithTx(tx))
		if err := txOrderExec.Create(ctx, &Order{ShopID: shopID1, Amount: amount}); err != nil {
			return err
		}
		return txOrderExec.Create(ctx, &Order{ShopID: shopID2, Amount: amount})
	}, dbhelper.WithManager(mgr), dbhelper.WithTxDatabaseGroupKey("order_dbs"))
	requireNoError(t, err)

	assertOrderExists(t, ctx, orderExec, shopID1, amount)
	assertOrderExists(t, ctx, orderExec, shopID2, amount)
}

func Test_Transaction_DbShardCrossShardRejected(t *testing.T) {
	ctx := context.Background()
	mgr := dbhelper.NewManager(dbspi.DatabaseConfig{
		DatabaseGroups: map[string]dbspi.DatabaseGroupConfig{
			dbspi.DefaultDatabaseGroupKey: defaultTestDatabaseGroupConfig(),
			"order_dbs": {
				Host: testDbHost, Port: testDbPort, User: testDbUser, Password: testDbPassword,
				DatabaseSharding: &dbspi.DatabaseShardingConfig{
					NameExpr:    "order_db_${idx}",
					ExpandExprs: []string{"${idx} := range(0, 2)", "${idx} = @{shop_id} % 2"},
				},
				TableSharding: &dbspi.TableShardingConfig{
					NameExpr:    "order_tab_${index}",
					ExpandExprs: []string{"${idx} := range(0, 10)", "${idx} = @{shop_id} % 10", "${index} = fill(${idx}, 8)"},
				},
			},
		},
	})

	txKey := dbspi.NewShardingKey().Set(OrderFields.ShopID, int64(12345))
	err := dbhelper.Transaction(ctx, func(tx *dbhelper.Tx) error {
		txOrderExec := dbhelper.NewExecutor(&Order{}, dbhelper.WithTx(tx))
		return txOrderExec.Create(ctx, &Order{ShopID: 12344, Amount: time.Now().UnixNano()})
	}, dbhelper.WithManager(mgr), dbhelper.WithTxDatabaseGroupKey("order_dbs"), dbhelper.WithTxShardingKey(txKey))
	requireErrorContains(t, err, "cross-shard")
}

func ensureShopTable(t *testing.T, ctx context.Context, mgr dbspi.Manager) {
	t.Helper()
	exec := dbhelper.NewExecutor(&User{}, dbhelper.WithManager(mgr))
	requireNoError(t, exec.Exec(ctx, `
CREATE TABLE IF NOT EXISTS shop_tab (
	id BIGINT NOT NULL AUTO_INCREMENT,
	name VARCHAR(255) NOT NULL DEFAULT '',
	owner_email VARCHAR(255) NOT NULL DEFAULT '',
	PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci`))
}

func cleanupTransactionRows(t *testing.T, ctx context.Context, userExec dbspi.Executor[*User], shopExec dbspi.Executor[*Shop], email string, shopName string) {
	t.Helper()
	requireNoError(t, userExec.DeleteByQuery(ctx, dbhelper.Q(NewUserFieldManager().Email.Eq(&email))))
	requireNoError(t, shopExec.DeleteByQuery(ctx, dbhelper.Q(ShopFields.Name.Eq(&shopName))))
}

func cleanupOrderRows(t *testing.T, ctx context.Context, orderExec dbspi.Executor[*Order], shopID int64, amount int64) {
	t.Helper()
	requireNoError(t, orderExec.DeleteByQuery(ctx, dbhelper.Q(
		OrderFields.ShopID.Eq(&shopID),
		dbhelper.NewField[int64]("amount").Eq(&amount),
	)))
}

func assertOrderExists(t *testing.T, ctx context.Context, orderExec dbspi.Executor[*Order], shopID int64, amount int64) {
	t.Helper()
	exists, _, err := orderExec.Exists(ctx, dbhelper.Q(
		OrderFields.ShopID.Eq(&shopID),
		dbhelper.NewField[int64]("amount").Eq(&amount),
	))
	requireNoError(t, err)
	if !exists {
		t.Fatalf("expected order row: shop_id=%d amount=%d", shopID, amount)
	}
}
