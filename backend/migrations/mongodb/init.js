// MongoDB initialization script
// Runs once on first container startup via docker-entrypoint-initdb.d

db = db.getSiblingDB('microservice_db');

// Create application user with least-privilege role
db.createUser({
  user: 'app_user',
  pwd:  process.env.MONGO_PASSWORD || 'changeme',
  roles: [
    { role: 'readWrite', db: 'microservice_db' },
  ],
});

// Create product_metadata collection with schema validation
db.createCollection('product_metadata', {
  validator: {
    $jsonSchema: {
      bsonType: 'object',
      required: ['product_id'],
      properties: {
        product_id: { bsonType: 'string', description: 'UUID of the product in PostgreSQL' },
        tags:       { bsonType: 'array',  items: { bsonType: 'string' } },
        attributes: { bsonType: 'object' },
        images:     { bsonType: 'array',  items: { bsonType: 'string' } },
        updated_at: { bsonType: 'date' },
      },
    },
  },
  validationLevel: 'moderate',
  validationAction: 'warn',
});

// Indexes (also created by the app at startup, but pre-creating is good practice)
db.product_metadata.createIndex({ product_id: 1 }, { unique: true, name: 'product_id_unique' });
db.product_metadata.createIndex({ tags: 1 },       { name: 'tags_idx' });
db.product_metadata.createIndex({ 'seo.slug': 1 }, { sparse: true, name: 'seo_slug_idx' });

// Audit log collection — capped (auto-purges old entries)
db.createCollection('audit_logs', {
  capped: true,
  size:   100 * 1024 * 1024, // 100 MB cap
  max:    500000,
});

db.audit_logs.createIndex({ entity_type: 1, entity_id: 1 });
db.audit_logs.createIndex({ actor_id: 1 });
db.audit_logs.createIndex({ created_at: -1 });

print('MongoDB initialization complete');
