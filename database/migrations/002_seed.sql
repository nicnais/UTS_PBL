INSERT INTO roles (name)
VALUES ('customer'), ('staff');

INSERT INTO permissions (name)
VALUES
    ('service:create'),
    ('service:update'),
    ('order:create'),
    ('order:read:any'),
    ('order:update:any');

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE
    (
        r.name = 'customer'
        AND p.name = 'order:create'
    )
    OR
    (
        r.name = 'staff'
        AND p.name IN (
            'service:create',
            'service:update',
            'order:read:any',
            'order:update:any'
        )
    );

INSERT INTO services (name, price_per_kg)
VALUES ('Cuci Setrika Reguler', 7000);