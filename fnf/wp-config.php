<?php

/**
 * WordPress Configuration File for MySQL
 */

// Database Settings
define('DB_NAME', 'fnf');
define('DB_USER', trim(getenv('WORDPRESS_DB_USER')));
define('DB_PASSWORD', trim(getenv('WORDPRESS_DB_PASSWORD')));
define('DB_HOST', 'mysql.mysql.svc.cluster.local');
define('DB_CHARSET', 'utf8');
define('DB_COLLATE', '');

// Table prefix
$table_prefix = 'fnf_';

// WordPress Language
define('WPLANG', '');

// Authentication Unique Keys and Salts
define('AUTH_KEY',         'rXvjMQAY8oYJwF6WU4hwGNwy3ZGx5hDc');
define('SECURE_AUTH_KEY',  'e3N6kp8wPXL4tBHcZfp9yRrGchmQ7KxB');
define('LOGGED_IN_KEY',    'qS9VwzFJA5G2Y3s7tRuMcLpaTnK8xBdv');
define('NONCE_KEY',        'k6zJwHfQmRc2P8NnEtU3VbX9yGvS5dZ7');
define('AUTH_SALT',        'zU9sHn3MjR5qX2pWfY7tGvKb4cT8dL6');
define('SECURE_AUTH_SALT', 'bP5mSfTnRv9gK2qZ6xL3wH8yG4jN7cD');
define('LOGGED_IN_SALT',   'mK4jH7gG2sD5fN9rB3pT8qZ6xW1vC0');
define('NONCE_SALT',       'rW5nB7tK2jF4qP9gM6sD3xH1cV8zL0');

// Debug Settings
define('WP_DEBUG', true);
define('WP_DEBUG_DISPLAY', true);
define('WP_DEBUG_LOG', true);

/* Sets up WordPress environment */
if (!defined('ABSPATH')) {
  define('ABSPATH', dirname(__FILE__) . '/');
}

/* Sets up WordPress vars and included files. */
require_once ABSPATH . 'wp-settings.php';
