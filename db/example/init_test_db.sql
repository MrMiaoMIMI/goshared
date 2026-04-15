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
-- Re-run this SQL before each test session to reset all data.
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
-- 1. Database: my_test
--    Used by: testNewDb() -> db_test.go, db_enhance_test.go,
--             db_sharding_test.go (most examples)
--    (already created above, USE is active)
-- ############################################################

-- 1.1 dbspi_test_user_tab (User entity)
CREATE TABLE `dbspi_test_user_tab` (
    `id`      BIGINT       NOT NULL AUTO_INCREMENT,
    `name`    VARCHAR(255) NOT NULL DEFAULT '',
    `email`   VARCHAR(255) NOT NULL DEFAULT '',
    `age`     INT          NOT NULL DEFAULT 0,
    `status`  VARCHAR(64)  NOT NULL DEFAULT '',
    `deleted` TINYINT(1)   NOT NULL DEFAULT 0,
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- 1.2 users table (for raw SQL query in Example 25)
CREATE TABLE `users` LIKE `dbspi_test_user_tab`;

-- 1.3 order_tab sharded tables (10-way, covers both 8-way and 10-way test configs)
CALL create_sharded_tables(
    'my_test', 'order_tab',
    '`id` BIGINT NOT NULL AUTO_INCREMENT, `shop_id` BIGINT NOT NULL DEFAULT 0, `amount` BIGINT NOT NULL DEFAULT 0, `status` INT NOT NULL DEFAULT 0, PRIMARY KEY (`id`)',
    10
);

-- 1.4 Seed data for User tests
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
--             Test_DbManager_Simple, Test_DbManager_DSN,
--             Test_DbManager_GlobalDefault
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
    `deleted` TINYINT(1)   NOT NULL DEFAULT 0,
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

INSERT INTO `dbspi_test_user_tab` (`id`, `name`, `email`, `age`, `status`, `deleted`) VALUES
    (1,  'Alice',   'alice@example.com',   25, 'active',   0),
    (13, 'Charlie', 'charlie@example.com', 28, 'active',   0),
    (14, 'David',   'david@example.com',   35, 'inactive', 0),
    (34, 'Bob',     'bob@example.com',     30, 'active',   0);


-- ############################################################
-- 3. Database: order_db
--    Used by: Config/Builder single-server tests
--             Test_Config_TableOnly, Test_Config_WithConnPool,
--             Test_Builder_TableOnly, Test_Builder_WithOptions
-- ############################################################
DROP DATABASE IF EXISTS `order_db`;
CREATE DATABASE `order_db` CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
USE `order_db`;

CALL `my_test`.create_sharded_tables(
    'order_db', 'order_tab',
    '`id` BIGINT NOT NULL AUTO_INCREMENT, `shop_id` BIGINT NOT NULL DEFAULT 0, `amount` BIGINT NOT NULL DEFAULT 0, `status` INT NOT NULL DEFAULT 0, PRIMARY KEY (`id`)',
    10
);


-- ############################################################
-- 4. Databases: order_db_0 ~ order_db_3
--    Used by: DB+Table sharding tests
--             Test_Config_DbAndTable, Test_Builder_DbAndTable,
--             Test_DbManager_ShardedWithReuse,
--             Test_DbManager_GlobalDefault
--    Each DB contains:
--      order_tab_00000000 ~ 00000009         (10 tables)
--      order_item_tab_00000000 ~ 00000009    (10 tables)
--      order_detail_tab_00000000 ~ 00000019  (20 tables)
-- ############################################################

-- --- order_db_0 ---
DROP DATABASE IF EXISTS `order_db_0`;
CREATE DATABASE `order_db_0` CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
USE `order_db_0`;

CALL `my_test`.create_sharded_tables(
    'order_db_0', 'order_tab',
    '`id` BIGINT NOT NULL AUTO_INCREMENT, `shop_id` BIGINT NOT NULL DEFAULT 0, `amount` BIGINT NOT NULL DEFAULT 0, `status` INT NOT NULL DEFAULT 0, PRIMARY KEY (`id`)',
    10
);
CALL `my_test`.create_sharded_tables(
    'order_db_0', 'order_item_tab',
    '`id` BIGINT NOT NULL AUTO_INCREMENT, `order_id` BIGINT NOT NULL DEFAULT 0, `shop_id` BIGINT NOT NULL DEFAULT 0, `name` VARCHAR(255) NOT NULL DEFAULT '''', PRIMARY KEY (`id`)',
    10
);
CALL `my_test`.create_sharded_tables(
    'order_db_0', 'order_detail_tab',
    '`id` BIGINT NOT NULL AUTO_INCREMENT, `order_id` BIGINT NOT NULL DEFAULT 0, `shop_id` BIGINT NOT NULL DEFAULT 0, `detail` TEXT, PRIMARY KEY (`id`)',
    20
);

-- --- order_db_1 ---
DROP DATABASE IF EXISTS `order_db_1`;
CREATE DATABASE `order_db_1` CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
USE `order_db_1`;

CALL `my_test`.create_sharded_tables(
    'order_db_1', 'order_tab',
    '`id` BIGINT NOT NULL AUTO_INCREMENT, `shop_id` BIGINT NOT NULL DEFAULT 0, `amount` BIGINT NOT NULL DEFAULT 0, `status` INT NOT NULL DEFAULT 0, PRIMARY KEY (`id`)',
    10
);
CALL `my_test`.create_sharded_tables(
    'order_db_1', 'order_item_tab',
    '`id` BIGINT NOT NULL AUTO_INCREMENT, `order_id` BIGINT NOT NULL DEFAULT 0, `shop_id` BIGINT NOT NULL DEFAULT 0, `name` VARCHAR(255) NOT NULL DEFAULT '''', PRIMARY KEY (`id`)',
    10
);
CALL `my_test`.create_sharded_tables(
    'order_db_1', 'order_detail_tab',
    '`id` BIGINT NOT NULL AUTO_INCREMENT, `order_id` BIGINT NOT NULL DEFAULT 0, `shop_id` BIGINT NOT NULL DEFAULT 0, `detail` TEXT, PRIMARY KEY (`id`)',
    20
);

-- --- order_db_2 ---
DROP DATABASE IF EXISTS `order_db_2`;
CREATE DATABASE `order_db_2` CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
USE `order_db_2`;

CALL `my_test`.create_sharded_tables(
    'order_db_2', 'order_tab',
    '`id` BIGINT NOT NULL AUTO_INCREMENT, `shop_id` BIGINT NOT NULL DEFAULT 0, `amount` BIGINT NOT NULL DEFAULT 0, `status` INT NOT NULL DEFAULT 0, PRIMARY KEY (`id`)',
    10
);
CALL `my_test`.create_sharded_tables(
    'order_db_2', 'order_item_tab',
    '`id` BIGINT NOT NULL AUTO_INCREMENT, `order_id` BIGINT NOT NULL DEFAULT 0, `shop_id` BIGINT NOT NULL DEFAULT 0, `name` VARCHAR(255) NOT NULL DEFAULT '''', PRIMARY KEY (`id`)',
    10
);
CALL `my_test`.create_sharded_tables(
    'order_db_2', 'order_detail_tab',
    '`id` BIGINT NOT NULL AUTO_INCREMENT, `order_id` BIGINT NOT NULL DEFAULT 0, `shop_id` BIGINT NOT NULL DEFAULT 0, `detail` TEXT, PRIMARY KEY (`id`)',
    20
);

-- --- order_db_3 ---
DROP DATABASE IF EXISTS `order_db_3`;
CREATE DATABASE `order_db_3` CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
USE `order_db_3`;

CALL `my_test`.create_sharded_tables(
    'order_db_3', 'order_tab',
    '`id` BIGINT NOT NULL AUTO_INCREMENT, `shop_id` BIGINT NOT NULL DEFAULT 0, `amount` BIGINT NOT NULL DEFAULT 0, `status` INT NOT NULL DEFAULT 0, PRIMARY KEY (`id`)',
    10
);
CALL `my_test`.create_sharded_tables(
    'order_db_3', 'order_item_tab',
    '`id` BIGINT NOT NULL AUTO_INCREMENT, `order_id` BIGINT NOT NULL DEFAULT 0, `shop_id` BIGINT NOT NULL DEFAULT 0, `name` VARCHAR(255) NOT NULL DEFAULT '''', PRIMARY KEY (`id`)',
    10
);
CALL `my_test`.create_sharded_tables(
    'order_db_3', 'order_detail_tab',
    '`id` BIGINT NOT NULL AUTO_INCREMENT, `order_id` BIGINT NOT NULL DEFAULT 0, `shop_id` BIGINT NOT NULL DEFAULT 0, `detail` TEXT, PRIMARY KEY (`id`)',
    20
);


-- ############################################################
-- 5. Databases: order_SG_db, order_TH_db, order_ID_db
--    Used by: Named DB sharding tests
--             Test_Config_NamedDbs, Test_Builder_NamedDbs,
--             Test_DbManager_NamedDbSharding
--    Each DB contains:
--      order_tab_00000000 ~ 00000009  (10 tables)
-- ############################################################

-- --- order_SG_db ---
DROP DATABASE IF EXISTS `order_SG_db`;
CREATE DATABASE `order_SG_db` CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
USE `order_SG_db`;

CALL `my_test`.create_sharded_tables(
    'order_SG_db', 'order_tab',
    '`id` BIGINT NOT NULL AUTO_INCREMENT, `shop_id` BIGINT NOT NULL DEFAULT 0, `amount` BIGINT NOT NULL DEFAULT 0, `status` INT NOT NULL DEFAULT 0, PRIMARY KEY (`id`)',
    10
);

-- --- order_TH_db ---
DROP DATABASE IF EXISTS `order_TH_db`;
CREATE DATABASE `order_TH_db` CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
USE `order_TH_db`;

CALL `my_test`.create_sharded_tables(
    'order_TH_db', 'order_tab',
    '`id` BIGINT NOT NULL AUTO_INCREMENT, `shop_id` BIGINT NOT NULL DEFAULT 0, `amount` BIGINT NOT NULL DEFAULT 0, `status` INT NOT NULL DEFAULT 0, PRIMARY KEY (`id`)',
    10
);

-- --- order_ID_db ---
DROP DATABASE IF EXISTS `order_ID_db`;
CREATE DATABASE `order_ID_db` CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
USE `order_ID_db`;

CALL `my_test`.create_sharded_tables(
    'order_ID_db', 'order_tab',
    '`id` BIGINT NOT NULL AUTO_INCREMENT, `shop_id` BIGINT NOT NULL DEFAULT 0, `amount` BIGINT NOT NULL DEFAULT 0, `status` INT NOT NULL DEFAULT 0, PRIMARY KEY (`id`)',
    10
);


-- ############################################################
-- Cleanup utility procedure
-- ############################################################
USE `my_test`;
DROP PROCEDURE IF EXISTS create_sharded_tables;


-- ============================================================
-- Summary of created resources
-- ============================================================
-- Databases (10):
--   my_test, my_app_db, order_db,
--   order_db_0, order_db_1, order_db_2, order_db_3,
--   order_SG_db, order_TH_db, order_ID_db
--
-- Tables per database:
--   my_test      : dbspi_test_user_tab, users, order_tab_* (x10)     = 12 tables
--   my_app_db    : dbspi_test_user_tab                                =  1 table
--   order_db     : order_tab_* (x10)                                  = 10 tables
--   order_db_0~3 : order_tab_*(x10) + order_item_tab_*(x10)
--                  + order_detail_tab_*(x20)                          = 40 tables x 4 = 160 tables
--   order_SG/TH/ID_db : order_tab_* (x10)                            = 10 tables x 3 = 30 tables
--                                                            Total   = 213 tables
--
-- All tests use the same local connection (127.0.0.1:3306).
-- ============================================================
