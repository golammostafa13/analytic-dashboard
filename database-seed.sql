-- Drop existing tables if they exist (for a fresh seed)
DROP TABLE IF EXISTS inventory_log, reviews, order_items, orders, categories, products, users CASCADE;

-- Create users table
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    username VARCHAR(50) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(100) NOT NULL,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    last_login BIGINT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    role VARCHAR(20) NOT NULL
);

-- Create categories table
CREATE TABLE categories (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE,
    description TEXT,
    parent_id BIGINT REFERENCES categories(id) ON DELETE SET NULL,
    created_at BIGINT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE
);

-- Create products table
CREATE TABLE products (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    price DECIMAL(10,2) NOT NULL,
    stock INTEGER NOT NULL,
    category_id BIGINT NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE
);

-- Create orders table
CREATE TABLE orders (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL,
    total_amount DECIMAL(12,2) NOT NULL,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    payment_status VARCHAR(20) NOT NULL
);

-- Create order_items table
CREATE TABLE order_items (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id BIGINT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    quantity INTEGER NOT NULL,
    unit_price DECIMAL(10,2) NOT NULL,
    created_at BIGINT NOT NULL
);

-- Create reviews table
CREATE TABLE reviews (
    id BIGSERIAL PRIMARY KEY,
    product_id BIGINT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    rating INTEGER NOT NULL CHECK (rating BETWEEN 1 AND 5),
    comment TEXT,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    is_verified BOOLEAN NOT NULL DEFAULT FALSE
);

-- Create inventory_log table
CREATE TABLE inventory_log (
    id BIGSERIAL PRIMARY KEY,
    product_id BIGINT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    quantity_change INTEGER NOT NULL,
    type VARCHAR(20) NOT NULL,
    reference_id BIGINT,
    created_at BIGINT NOT NULL,
    created_by BIGINT NOT NULL REFERENCES users(id)
);

-- Insert dummy users
INSERT INTO users (email, username, password_hash, full_name, created_at, updated_at, is_active, role)
VALUES 
('admin@example.com', 'admin', 'hashed_password', 'Admin User', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT, TRUE, 'admin'),
('user@example.com', 'john_doe', 'hashed_password', 'John Doe', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT, TRUE, 'user');

-- Insert dummy categories
INSERT INTO categories (name, description, created_at, is_active)
VALUES 
('Electronics', 'Electronic gadgets and accessories', EXTRACT(EPOCH FROM NOW())::BIGINT, TRUE),
('Books', 'Books of various genres', EXTRACT(EPOCH FROM NOW())::BIGINT, TRUE);

-- Insert dummy products
INSERT INTO products (name, description, price, stock, category_id, created_at, updated_at, is_active)
VALUES 
('Laptop', 'High-performance laptop', 1200.99, 10, 1, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT, TRUE),
('Smartphone', 'Latest model smartphone', 899.99, 20, 1, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT, TRUE),
('Programming Book', 'Learn Go programming', 39.99, 50, 2, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT, TRUE);

-- Insert dummy orders
INSERT INTO orders (user_id, status, total_amount, created_at, updated_at, payment_status)
VALUES 
(2, 'pending', 1200.99, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT, 'unpaid');

-- Insert dummy order items
INSERT INTO order_items (order_id, product_id, quantity, unit_price, created_at)
VALUES 
(1, 1, 1, 1200.99, EXTRACT(EPOCH FROM NOW())::BIGINT);

-- Insert dummy reviews
INSERT INTO reviews (product_id, user_id, rating, comment, created_at, updated_at, is_verified)
VALUES 
(1, 2, 5, 'Great laptop!', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT, TRUE);

-- Insert dummy inventory logs
INSERT INTO inventory_log (product_id, quantity_change, type, reference_id, created_at, created_by)
VALUES 
(1, -1, 'order', 1, EXTRACT(EPOCH FROM NOW())::BIGINT, 2);
