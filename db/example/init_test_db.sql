-- ============================================================
-- Database Initialization SQL for db/example Unit Tests
-- ============================================================
--
-- Prerequisites:
--   MySQL server running on 127.0.0.1:3306
--
-- Credentials (unified across all tests):
--   User: root / Password: 123456
--   Host: 127.0.0.1 / Port: 3306
--   (defined as constants in db_test.go)
--
-- Usage:
--   mysql -u root -p123456 < db/example/init_test_db.sql
--
-- This SQL is fully idempotent — safe to re-run at any time.
-- All databases are dropped and recreated from scratch.
-- ============================================================


-- ############################################################
-- 0. Bootstrap: create my_test first to host the utility procedure
-- ############################################################
DROP DATABASE IF EXISTS `my_test`;
CREATE DATABASE `my_test` CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
USE `my_test`;


-- ============================================================
-- Utility: stored procedure to batch-create sharded tables
-- ============================================================
DROP PROCEDURE IF EXISTS create_sharded_tables;
DELIMITER //
CREATE PROCEDURE create_sharded_tables(
    IN db_name    VARCHAR(64),
    IN base_table VARCHAR(64),
    IN col_defs   TEXT,
    IN shard_cnt  INT
)
BEGIN
    DECLARE i INT DEFAULT 0;
    DECLARE tbl_name VARCHAR(128);
    DECLARE full_sql TEXT;

    WHILE i < shard_cnt DO
        SET tbl_name = CONCAT(base_table, '_', LPAD(i, 8, '0'));
        SET full_sql = CONCAT(
            'CREATE TABLE `', db_name, '`.`', tbl_name, '` (', col_defs, ') ',
            'ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci'
        );
        SET @ddl = full_sql;
        PREPARE stmt FROM @ddl;
        EXECUTE stmt;
        DEALLOCATE PREPARE stmt;
        SET i = i + 1;
    END WHILE;
END //
DELIMITER ;


-- ############################################################
-- Column definitions (reused across databases)
-- ############################################################
-- order_tab includes `region` for RegionalOrder composite-key tests.
-- The Order struct simply ignores this column (GORM only maps declared fields).
SET @order_cols = '`id` BIGINT NOT NULL AUTO_INCREMENT, `shop_id` BIGINT NOT NULL DEFAULT 0, `region` VARCHAR(64) NOT NULL DEFAULT '''', `amount` BIGINT NOT NULL DEFAULT 0, `status` INT NOT NULL DEFAULT 0, PRIMARY KEY (`id`)';

SET @order_item_cols = '`id` BIGINT NOT NULL AUTO_INCREMENT, `order_id` BIGINT NOT NULL DEFAULT 0, `shop_id` BIGINT NOT NULL DEFAULT 0, `name` VARCHAR(255) NOT NULL DEFAULT '''', PRIMARY KEY (`id`)';

SET @order_detail_cols = '`id` BIGINT NOT NULL AUTO_INCREMENT, `order_id` BIGINT NOT NULL DEFAULT 0, `shop_id` BIGINT NOT NULL DEFAULT 0, `detail` TEXT, PRIMARY KEY (`id`)';


-- ############################################################
-- 1. Database: my_test
--    Used by: testDbManager() in all test files, plus
--             Config/Builder single-server tests (table-only sharding)
-- ############################################################

-- 1.1 dbspi_test_user_tab (User entity)
CREATE TABLE `dbspi_test_user_tab` (
    `id`      BIGINT       NOT NULL AUTO_INCREMENT,
    `name`    VARCHAR(255) NOT NULL DEFAULT '',
    `email`   VARCHAR(255) NOT NULL DEFAULT '',
    `age`     INT          NOT NULL DEFAULT 0,
    `status`  VARCHAR(64)  NOT NULL DEFAULT '',
    `creator` VARCHAR(255) NOT NULL DEFAULT '',
    `updater` VARCHAR(255) NOT NULL DEFAULT '',
    `ctime`   BIGINT       NOT NULL DEFAULT 0,
    `mtime`   BIGINT       NOT NULL DEFAULT 0,
    `deleted` TINYINT(1)   NOT NULL DEFAULT 0,
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- 1.2 users table (for raw SQL query in Example 25)
CREATE TABLE `users` LIKE `dbspi_test_user_tab`;

-- 1.3 order_tab base table (non-sharded, used by region-based DB-only sharding tests)
CREATE TABLE `order_tab` (
    `id`      BIGINT       NOT NULL AUTO_INCREMENT,
    `shop_id` BIGINT       NOT NULL DEFAULT 0,
    `region`  VARCHAR(64)  NOT NULL DEFAULT '',
    `amount`  BIGINT       NOT NULL DEFAULT 0,
    `status`  INT          NOT NULL DEFAULT 0,
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- 1.4 order_tab sharded tables (10-way)
CALL create_sharded_tables('my_test', 'order_tab', @order_cols, 10);

-- 1.5 order_detail_tab sharded tables (10-way, for ${table} variable tests)
CALL create_sharded_tables('my_test', 'order_detail_tab', @order_detail_cols, 10);

-- 1.6 Seed data for User tests
INSERT INTO `dbspi_test_user_tab` (`id`, `name`, `email`, `age`, `status`, `deleted`) VALUES
    (1,  'Alice',   'alice@example.com',   25, 'active',   0),
    (13, 'Charlie', 'charlie@example.com', 28, 'active',   0),
    (14, 'David',   'david@example.com',   35, 'inactive', 0),
    (34, 'Bob',     'bob@example.com',     30, 'active',   0);

INSERT INTO `users` (`id`, `name`, `email`, `age`, `status`, `deleted`) VALUES
    (1,  'Alice',   'alice@example.com',   25, 'active',   0),
    (13, 'Charlie', 'charlie@example.com', 28, 'active',   0),
    (14, 'David',   'david@example.com',   35, 'inactive', 0),
    (34, 'Bob',     'bob@example.com',     30, 'active',   0);


-- ############################################################
-- 2. Database: my_app_db
--    Used by: DbManager tests (default entry)
--    Kept separate from my_test for data isolation —
--    other tests modify my_test.dbspi_test_user_tab.
-- ############################################################
DROP DATABASE IF EXISTS `my_app_db`;
CREATE DATABASE `my_app_db` CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
USE `my_app_db`;

CREATE TABLE `dbspi_test_user_tab` (
    `id`      BIGINT       NOT NULL AUTO_INCREMENT,
    `name`    VARCHAR(255) NOT NULL DEFAULT '',
    `email`   VARCHAR(255) NOT NULL DEFAULT '',
    `age`     INT          NOT NULL DEFAULT 0,
    `status`  VARCHAR(64)  NOT NULL DEFAULT '',
    `creator` VARCHAR(255) NOT NULL DEFAULT '',
    `updater` VARCHAR(255) NOT NULL DEFAULT '',
    `ctime`   BIGINT       NOT NULL DEFAULT 0,
    `mtime`   BIGINT       NOT NULL DEFAULT 0,
    `deleted` TINYINT(1)   NOT NULL DEFAULT 0,
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

INSERT INTO `dbspi_test_user_tab` (`id`, `name`, `email`, `age`, `status`, `deleted`) VALUES
    (1,  'Alice',   'alice@example.com',   25, 'active',   0),
    (13, 'Charlie', 'charlie@example.com', 28, 'active',   0),
    (14, 'David',   'david@example.com',   35, 'inactive', 0),
    (34, 'Bob',     'bob@example.com',     30, 'active',   0);


-- ############################################################
-- 3. Databases: order_db_0, order_db_1
--    Used by: DB+Table sharding tests, MultiServer tests,
--             DbManager sharded entries
--    Each DB contains:
--      order_tab_00000000 ~ 00000009         (10 tables)
--      order_item_tab_00000000 ~ 00000009    (10 tables)
--      order_detail_tab_00000000 ~ 00000009  (10 tables)
-- ############################################################

-- --- order_db_0 ---
DROP DATABASE IF EXISTS `order_db_0`;
CREATE DATABASE `order_db_0` CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
USE `order_db_0`;

CALL `my_test`.create_sharded_tables('order_db_0', 'order_tab', @order_cols, 10);
CALL `my_test`.create_sharded_tables('order_db_0', 'order_item_tab', @order_item_cols, 10);
CALL `my_test`.create_sharded_tables('order_db_0', 'order_detail_tab', @order_detail_cols, 10);

-- --- order_db_1 ---
DROP DATABASE IF EXISTS `order_db_1`;
CREATE DATABASE `order_db_1` CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
USE `order_db_1`;

CALL `my_test`.create_sharded_tables('order_db_1', 'order_tab', @order_cols, 10);
CALL `my_test`.create_sharded_tables('order_db_1', 'order_item_tab', @order_item_cols, 10);
CALL `my_test`.create_sharded_tables('order_db_1', 'order_detail_tab', @order_detail_cols, 10);


-- ############################################################
-- 4. Databases: order_SG_db, order_TH_db
--    Used by: Named DB sharding tests (region-based)
--    Each DB contains:
--      order_tab_00000000 ~ 00000009  (10 tables)
-- ############################################################

-- --- order_SG_db ---
DROP DATABASE IF EXISTS `order_SG_db`;
CREATE DATABASE `order_SG_db` CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
USE `order_SG_db`;

CALL `my_test`.create_sharded_tables('order_SG_db', 'order_tab', @order_cols, 10);

-- --- order_TH_db ---
DROP DATABASE IF EXISTS `order_TH_db`;
CREATE DATABASE `order_TH_db` CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
USE `order_TH_db`;

CALL `my_test`.create_sharded_tables('order_TH_db', 'order_tab', @order_cols, 10);


-- ############################################################
-- Cleanup: drop databases that are no longer used
-- (safe for fresh installs; prevents stale leftovers on re-run)
-- ############################################################
DROP DATABASE IF EXISTS `order_db`;
DROP DATABASE IF EXISTS `order_db_2`;
DROP DATABASE IF EXISTS `order_db_3`;
DROP DATABASE IF EXISTS `order_ID_db`;


-- ############################################################
-- Cleanup utility procedure
-- ############################################################
USE `my_test`;
DROP PROCEDURE IF EXISTS create_sharded_tables;


-- ============================================================
-- Summary of created resources
-- ============================================================
-- Databases (6):
--   my_test, my_app_db,
--   order_db_0, order_db_1,
--   order_SG_db, order_TH_db
--
-- Tables per database:
--   my_test      : dbspi_test_user_tab, users, order_tab (base),
--                  order_tab_* (x10),
--                  order_detail_tab_* (x10)                       = 23 tables
--   my_app_db    : dbspi_test_user_tab                            =  1 table
--   order_db_0~1 : order_tab_*(x10) + order_item_tab_*(x10)
--                  + order_detail_tab_*(x10)                      = 30 tables x 2 = 60 tables
--   order_SG/TH_db : order_tab_* (x10)                           = 10 tables x 2 = 20 tables
--                                                        Total   = 104 tables
--
-- All order_tab tables include a `region` column for RegionalOrder tests.
-- order_detail_tab_* in my_test is for ${table} variable tests (OrderDetailTab entity).
-- All tests use the same local connection (127.0.0.1:3306).
-- ============================================================
