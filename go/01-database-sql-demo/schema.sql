-- 金额以“分”为单位保存为整数。订单项保留成交单价，避免商品调价篡改历史订单。
CREATE TABLE products (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    sku         VARCHAR(64) NOT NULL,
    name        VARCHAR(128) NOT NULL,
    price_cents BIGINT UNSIGNED NOT NULL,
    stock       BIGINT UNSIGNED NOT NULL,
    created_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uk_products_sku (sku)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE orders (
    id                 BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    customer_id        BIGINT UNSIGNED NOT NULL,
    status             VARCHAR(32) NOT NULL,
    total_amount_cents BIGINT UNSIGNED NOT NULL,
    created_at         DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE order_items (
    id               BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    order_id         BIGINT UNSIGNED NOT NULL,
    product_id       BIGINT UNSIGNED NOT NULL,
    quantity         BIGINT UNSIGNED NOT NULL,
    unit_price_cents BIGINT UNSIGNED NOT NULL,
    PRIMARY KEY (id),
    KEY idx_order_items_order_id (order_id),
    CONSTRAINT fk_order_items_order FOREIGN KEY (order_id) REFERENCES orders(id),
    CONSTRAINT fk_order_items_product FOREIGN KEY (product_id) REFERENCES products(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
