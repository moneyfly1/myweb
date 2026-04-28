-- 创建营销活动参与记录表
CREATE TABLE IF NOT EXISTS promotion_participations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    promotion_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    order_id INTEGER,
    reward_type VARCHAR(50) NOT NULL,
    reward_value DECIMAL(10,2) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    applied_at DATETIME,
    expire_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (promotion_id) REFERENCES promotions(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_promotion_participations_promotion_id ON promotion_participations(promotion_id);
CREATE INDEX IF NOT EXISTS idx_promotion_participations_user_id ON promotion_participations(user_id);
CREATE INDEX IF NOT EXISTS idx_promotion_participations_order_id ON promotion_participations(order_id);
CREATE INDEX IF NOT EXISTS idx_promotion_participations_status ON promotion_participations(status);
