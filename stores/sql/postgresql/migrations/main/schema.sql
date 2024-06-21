-- dbArchivedContract
CREATE TABLE archived_contracts (
  id SERIAL PRIMARY KEY,
  created_at timestamp DEFAULT NULL,
  fcid int NOT NULL UNIQUE,
  renewed_from int DEFAULT NULL,
  contract_price text,
  state smallint NOT NULL DEFAULT 0,
  total_cost text,
  proof_height int DEFAULT '0',
  revision_height int DEFAULT '0',
  revision_number varchar(191) NOT NULL DEFAULT '0',
  size int DEFAULT NULL,
  start_height int NOT NULL,
  window_start int NOT NULL DEFAULT '0',
  window_end int NOT NULL DEFAULT '0',
  upload_spending text,
  download_spending text,
  fund_account_spending text,
  delete_spending text,
  list_spending text,
  renewed_to int DEFAULT NULL,
  host int NOT NULL,
  reason text
);
CREATE INDEX idx_archived_contracts_renewed_from ON archived_contracts (renewed_from);
CREATE INDEX idx_archived_contracts_proof_height ON archived_contracts (proof_height);
CREATE INDEX idx_archived_contracts_revision_height ON archived_contracts (revision_height);
CREATE INDEX idx_archived_contracts_start_height ON archived_contracts (start_height);
CREATE INDEX idx_archived_contracts_host ON archived_contracts (host);
CREATE INDEX idx_archived_contracts_fc_id ON archived_contracts (fcid);
CREATE INDEX idx_archived_contracts_state ON archived_contracts (state);
CREATE INDEX idx_archived_contracts_window_start ON archived_contracts (window_start);
CREATE INDEX idx_archived_contracts_window_end ON archived_contracts (window_end);
CREATE INDEX idx_archived_contracts_renewed_to ON archived_contracts (renewed_to);


-- dbAutopilot
CREATE TABLE autopilots (
  id SERIAL PRIMARY KEY,
  created_at timestamp DEFAULT NULL,
  identifier varchar(191) NOT NULL UNIQUE,
  config text,
  current_period int DEFAULT '0'
);

-- dbBucket
CREATE TABLE buckets (
  id SERIAL PRIMARY KEY,
  created_at timestamp DEFAULT NULL,
  policy JSONB,
  name varchar(255) DEFAULT NULL UNIQUE
);
CREATE INDEX idx_buckets_name ON buckets (name);

-- dbBufferedSlab
CREATE TABLE buffered_slabs (
  id SERIAL PRIMARY KEY,
  created_at timestamp DEFAULT NULL,
  filename text
);

-- dbConsensusInfo
CREATE TABLE consensus_infos (
  id SERIAL PRIMARY KEY,
  created_at timestamp DEFAULT NULL,
  cc_id bytea,
  height int DEFAULT NULL,
  block_id bytea
);

-- dbHost
CREATE TABLE hosts (
  id SERIAL PRIMARY KEY,
  created_at timestamp DEFAULT NULL,
  public_key int NOT NULL UNIQUE,
  settings JSONB,
  price_table text,
  price_table_expiry timestamp DEFAULT NULL,
  total_scans int DEFAULT NULL,
  last_scan int DEFAULT NULL,
  last_scan_success boolean DEFAULT NULL,
  second_to_last_scan_success boolean DEFAULT NULL,
  scanned boolean DEFAULT NULL,
  uptime int DEFAULT NULL,
  downtime int DEFAULT NULL,
  recent_downtime int DEFAULT NULL,
  recent_scan_failures int DEFAULT NULL,
  successful_interactions double precision DEFAULT NULL,
  failed_interactions double precision DEFAULT NULL,
  lost_sectors int DEFAULT NULL,
  last_announcement timestamp DEFAULT NULL,
  net_address varchar(191) DEFAULT NULL
);
CREATE INDEX idx_hosts_public_key ON hosts (public_key);
CREATE INDEX idx_hosts_last_scan ON hosts (last_scan);
CREATE INDEX idx_hosts_scanned ON hosts (scanned);
CREATE INDEX idx_hosts_recent_downtime ON hosts (recent_downtime);
CREATE INDEX idx_hosts_recent_scan_failures ON hosts (recent_scan_failures);
CREATE INDEX idx_hosts_net_address ON hosts (net_address);

-- dbContract
CREATE TABLE contracts (
  id SERIAL PRIMARY KEY,
  created_at timestamp DEFAULT NULL,
  fcid int NOT NULL UNIQUE,
  renewed_from int DEFAULT NULL,
  contract_price text,
  state smallint NOT NULL DEFAULT '0',
  total_cost text,
  proof_height int DEFAULT '0',
  revision_height int DEFAULT '0',
  revision_number varchar(191) NOT NULL DEFAULT '0',
  size int DEFAULT NULL,
  start_height int NOT NULL,
  window_start int NOT NULL DEFAULT '0',
  window_end int NOT NULL DEFAULT '0',
  upload_spending text,
  download_spending text,
  fund_account_spending text,
  delete_spending text,
  list_spending text,
  host_id int DEFAULT NULL,
  CONSTRAINT fk_contracts_host FOREIGN KEY (host_id) REFERENCES hosts (id)
);
CREATE INDEX idx_contracts_window_end ON contracts (window_end);
CREATE INDEX idx_contracts_host_id ON contracts (host_id);
CREATE INDEX idx_contracts_renewed_from ON contracts (renewed_from);
CREATE INDEX idx_contracts_state ON contracts (state);
CREATE INDEX idx_contracts_proof_height ON contracts (proof_height);
CREATE INDEX idx_contracts_start_height ON contracts (start_height);
CREATE INDEX idx_contracts_fc_id ON contracts (fcid);
CREATE INDEX idx_contracts_revision_height ON contracts (revision_height);
CREATE INDEX idx_contracts_window_start ON contracts (window_start);

-- dbContractSet
CREATE TABLE contract_sets (
  id SERIAL PRIMARY KEY,
  created_at timestamp DEFAULT NULL,
  name varchar(191) DEFAULT NULL UNIQUE
);
CREATE INDEX idx_contract_sets_name ON contract_sets (name);

-- dbSlab
CREATE TABLE slabs (
  id SERIAL PRIMARY KEY,
  created_at timestamp DEFAULT NULL,
  db_contract_set_id int DEFAULT NULL,
  db_buffered_slab_id int DEFAULT NULL,
  health double precision NOT NULL DEFAULT '1',
  health_valid_until int NOT NULL DEFAULT '0',
  key int NOT NULL UNIQUE,
  min_shards smallint DEFAULT NULL,
  total_shards smallint DEFAULT NULL,
  CONSTRAINT fk_buffered_slabs_db_slab FOREIGN KEY (db_buffered_slab_id) REFERENCES buffered_slabs (id),
  CONSTRAINT fk_slabs_db_contract_set FOREIGN KEY (db_contract_set_id) REFERENCES contract_sets (id)
);
CREATE INDEX idx_slabs_min_shards ON slabs (min_shards);
CREATE INDEX idx_slabs_total_shards ON slabs (total_shards);
CREATE INDEX idx_slabs_db_contract_set_id ON slabs (db_contract_set_id);
CREATE INDEX idx_slabs_db_buffered_slab_id ON slabs (db_buffered_slab_id);
CREATE INDEX idx_slabs_health ON slabs (health);
CREATE INDEX idx_slabs_health_valid_until ON slabs (health_valid_until);

-- dbSector
CREATE TABLE sectors (
  id SERIAL PRIMARY KEY,
  created_at timestamp DEFAULT NULL,
  db_slab_id int NOT NULL,
  slab_index int NOT NULL,
  latest_host bytea NOT NULL,
  root int NOT NULL UNIQUE,
  CONSTRAINT fk_slabs_shards FOREIGN KEY (db_slab_id) REFERENCES slabs (id) ON DELETE CASCADE
);
CREATE INDEX idx_sectors_db_slab_id ON sectors (db_slab_id);
CREATE INDEX idx_sectors_slab_index ON sectors (slab_index);
CREATE INDEX idx_sectors_root ON sectors (root);
CREATE UNIQUE INDEX idx_slab_id_slab_index ON sectors (db_slab_id, slab_index);

-- dbContract <-> dbSector
CREATE TABLE contract_sectors (
  db_sector_id int NOT NULL,
  db_contract_id int NOT NULL,
  PRIMARY KEY (db_sector_id,db_contract_id),
  CONSTRAINT fk_contract_sectors_db_contract FOREIGN KEY (db_contract_id) REFERENCES contracts (id) ON DELETE CASCADE,
  CONSTRAINT fk_contract_sectors_db_sector FOREIGN KEY (db_sector_id) REFERENCES sectors (id) ON DELETE CASCADE
);
CREATE INDEX idx_contract_sectors_db_sector_id ON contract_sectors (db_sector_id);
CREATE INDEX idx_contract_sectors_db_contract_id ON contract_sectors (db_contract_id);

-- dbContractSet <-> dbContract
CREATE TABLE contract_set_contracts (
  db_contract_set_id int NOT NULL,
  db_contract_id int NOT NULL,
  PRIMARY KEY (db_contract_set_id, db_contract_id),
  FOREIGN KEY (db_contract_id) REFERENCES contracts (id) ON DELETE CASCADE,
  FOREIGN KEY (db_contract_set_id) REFERENCES contract_sets (id) ON DELETE CASCADE
);


CREATE INDEX idx_contract_set_contracts_db_contract_id ON contract_set_contracts (db_contract_id);

-- dbAccount
CREATE TABLE ephemeral_accounts (
  id SERIAL PRIMARY KEY,
  created_at timestamp DEFAULT NULL,
  account_id int NOT NULL UNIQUE,
  clean_shutdown boolean DEFAULT '0',
  host bytea NOT NULL,
  balance text,
  drift text,
  requires_sync boolean DEFAULT NULL
);
CREATE INDEX idx_ephemeral_accounts_requires_sync ON ephemeral_accounts (requires_sync);

-- dbAllowlistEntry
CREATE TABLE host_allowlist_entries (
  id SERIAL PRIMARY KEY,
  created_at timestamp DEFAULT NULL,
  entry int NOT NULL UNIQUE
);
CREATE INDEX idx_host_allowlist_entries_entry ON host_allowlist_entries (entry);

-- dbAllowlistEntry <-> dbHost
CREATE TABLE host_allowlist_entry_hosts (
  db_allowlist_entry_id int NOT NULL,
  db_host_id int NOT NULL,
  PRIMARY KEY (db_allowlist_entry_id,db_host_id),
  CONSTRAINT fk_host_allowlist_entry_hosts_db_allowlist_entry FOREIGN KEY (db_allowlist_entry_id) REFERENCES host_allowlist_entries (id) ON DELETE CASCADE,
  CONSTRAINT fk_host_allowlist_entry_hosts_db_host FOREIGN KEY (db_host_id) REFERENCES hosts (id) ON DELETE CASCADE
);
CREATE INDEX idx_host_allowlist_entry_hosts_db_host_id ON host_allowlist_entry_hosts (db_host_id);

-- dbHostAnnouncement
CREATE TABLE host_announcements (
  id SERIAL PRIMARY KEY,
  created_at timestamp DEFAULT NULL,
  host_key bytea NOT NULL,
  block_height int DEFAULT NULL,
  block_id text,
  net_address text
);

-- dbBlocklistEntry
CREATE TABLE host_blocklist_entries (
  id SERIAL PRIMARY KEY,
  created_at timestamp DEFAULT NULL,
  entry varchar(191) NOT NULL UNIQUE
);
CREATE INDEX idx_host_blocklist_entries_entry ON host_blocklist_entries (entry);

-- dbBlocklistEntry <-> dbHost
CREATE TABLE host_blocklist_entry_hosts (
  db_blocklist_entry_id int NOT NULL,
  db_host_id int NOT NULL,
  PRIMARY KEY (db_blocklist_entry_id,db_host_id),
  CONSTRAINT fk_host_blocklist_entry_hosts_db_blocklist_entry FOREIGN KEY (db_blocklist_entry_id) REFERENCES host_blocklist_entries (id) ON DELETE CASCADE,
  CONSTRAINT fk_host_blocklist_entry_hosts_db_host FOREIGN KEY (db_host_id) REFERENCES hosts (id) ON DELETE CASCADE
);
CREATE INDEX idx_host_blocklist_entry_hosts_db_host_id ON host_blocklist_entry_hosts (db_host_id);

-- dbMultipartUpload
CREATE TABLE multipart_uploads (
  id SERIAL PRIMARY KEY,
  created_at timestamp DEFAULT NULL,
  key bytea,
  upload_id varchar(64) NOT NULL UNIQUE,
  object_id varchar(766) DEFAULT NULL,
  db_bucket_id int NOT NULL,
  mime_type varchar(191) DEFAULT NULL,
  CONSTRAINT fk_multipart_uploads_db_bucket FOREIGN KEY (db_bucket_id) REFERENCES buckets (id) ON DELETE CASCADE
);
CREATE INDEX idx_multipart_uploads_object_id ON multipart_uploads (object_id);
CREATE INDEX idx_multipart_uploads_db_bucket_id ON multipart_uploads (db_bucket_id);
CREATE INDEX idx_multipart_uploads_mime_type ON multipart_uploads (mime_type);

-- dbMultipartPart
CREATE TABLE multipart_parts (
  id SERIAL PRIMARY KEY,
  created_at timestamp DEFAULT NULL,
  etag varchar(191) DEFAULT NULL,
  part_number int DEFAULT NULL,
  size int DEFAULT NULL,
  db_multipart_upload_id int NOT NULL,
  CONSTRAINT fk_multipart_uploads_parts FOREIGN KEY (db_multipart_upload_id) REFERENCES multipart_uploads (id) ON DELETE CASCADE
);
CREATE INDEX idx_multipart_parts_etag ON multipart_parts (etag);
CREATE INDEX idx_multipart_parts_part_number ON multipart_parts (part_number);
CREATE INDEX idx_multipart_parts_db_multipart_upload_id ON multipart_parts (db_multipart_upload_id);

-- dbDirectory
CREATE TABLE directories (
  id SERIAL PRIMARY KEY,
  created_at timestamp DEFAULT NULL,
  db_parent_id int,
  name varchar(766) DEFAULT NULL UNIQUE,
  CONSTRAINT fk_directories_db_directories FOREIGN KEY (db_parent_id) REFERENCES directories (id) ON DELETE CASCADE
);
CREATE INDEX idx_directories_parent_id ON directories (db_parent_id);

-- dbObject
CREATE TABLE objects (
  id SERIAL PRIMARY KEY,
  created_at timestamp DEFAULT NULL,
  db_bucket_id int NOT NULL,
  db_directory_id int NOT NULL,
  object_id varchar(766) DEFAULT NULL,
  key bytea,
  health double precision NOT NULL DEFAULT '1',
  size int DEFAULT NULL,
  mime_type text,
  etag varchar(191) DEFAULT NULL,
  CONSTRAINT fk_objects_db_bucket FOREIGN KEY (db_bucket_id) REFERENCES buckets (id),
  CONSTRAINT fk_objects_db_directory_id FOREIGN KEY (db_directory_id) REFERENCES directories (id)
);
CREATE INDEX idx_objects_db_bucket_id ON objects (db_bucket_id);
CREATE INDEX idx_objects_object_id ON objects (object_id);
CREATE INDEX idx_objects_health ON objects (health);
CREATE INDEX idx_objects_etag ON objects (etag);
CREATE INDEX idx_objects_size ON objects (size);
CREATE INDEX idx_objects_created_at ON objects (created_at);
CREATE INDEX idx_objects_db_directory_id ON objects (db_directory_id);
CREATE UNIQUE INDEX idx_objects_bid_oid ON objects (db_bucket_id, object_id);

-- dbSetting
CREATE TABLE settings (
  id SERIAL PRIMARY KEY,
  created_at timestamp DEFAULT NULL,
  key varchar(191) NOT NULL UNIQUE,
  value text NOT NULL
);
CREATE INDEX idx_settings_key ON settings (key);

-- dbSiacoinElement
CREATE TABLE siacoin_elements (
  id SERIAL PRIMARY KEY,
  created_at timestamp DEFAULT NULL,
  value text,
  address int DEFAULT NULL,
  output_id int NOT NULL UNIQUE,
  maturity_height int DEFAULT NULL
);
CREATE INDEX idx_siacoin_elements_output_id ON siacoin_elements (output_id);
CREATE INDEX idx_siacoin_elements_maturity_height ON siacoin_elements (maturity_height);

-- dbSlice
CREATE TABLE slices (
  id SERIAL PRIMARY KEY,
  created_at timestamp DEFAULT NULL,
  db_object_id int DEFAULT NULL,
  object_index int DEFAULT NULL,
  db_multipart_part_id int DEFAULT NULL,
  db_slab_id int DEFAULT NULL,
  "offset" int DEFAULT NULL,
  length int DEFAULT NULL,
  CONSTRAINT fk_multipart_parts_slabs FOREIGN KEY (db_multipart_part_id) REFERENCES multipart_parts (id) ON DELETE CASCADE,
  CONSTRAINT fk_objects_slabs FOREIGN KEY (db_object_id) REFERENCES objects (id) ON DELETE CASCADE,
  CONSTRAINT fk_slabs_slices FOREIGN KEY (db_slab_id) REFERENCES slabs (id)
);
CREATE INDEX idx_slices_db_object_id ON slices (db_object_id);
CREATE INDEX idx_slices_object_index ON slices (object_index);
CREATE INDEX idx_slices_db_multipart_part_id ON slices (db_multipart_part_id);
CREATE INDEX idx_slices_db_slab_id ON slices (db_slab_id);

-- dbTransaction
CREATE TABLE transactions (
  id SERIAL PRIMARY KEY,
  created_at timestamp DEFAULT NULL,
  raw text,
  height int DEFAULT NULL,
  block_id int DEFAULT NULL,
  transaction_id int NOT NULL UNIQUE,
  inflow text,
  outflow text,
  timestamp int DEFAULT NULL
);
CREATE INDEX idx_transactions_transaction_id ON transactions (transaction_id);
CREATE INDEX idx_transactions_timestamp ON transactions (timestamp);

-- dbWebhook
CREATE TABLE webhooks (
  id SERIAL PRIMARY KEY,
  created_at timestamp DEFAULT NULL,
  module varchar(255) NOT NULL,
  event varchar(255) NOT NULL,
  url varchar(255) NOT NULL,
  headers JSONB DEFAULT ('{}'),
  UNIQUE (module,event,url)
);

-- dbObjectUserMetadata
CREATE TABLE object_user_metadata (
  id SERIAL PRIMARY KEY,
  created_at timestamp DEFAULT NULL,
  db_object_id int DEFAULT NULL,
  db_multipart_upload_id int DEFAULT NULL,
  key varchar(255) DEFAULT NULL,
  value text,
  UNIQUE (db_object_id, db_multipart_upload_id, key),
  CONSTRAINT fk_object_user_metadata FOREIGN KEY (db_object_id) REFERENCES objects (id) ON DELETE CASCADE,
  CONSTRAINT fk_multipart_upload_user_metadata FOREIGN KEY (db_multipart_upload_id) REFERENCES multipart_uploads (id) ON DELETE SET NULL
);

-- dbHostCheck
CREATE TABLE host_checks (
  id SERIAL PRIMARY KEY,
  created_at timestamp DEFAULT NULL,

  db_autopilot_id int NOT NULL,
  db_host_id int NOT NULL,

  usability_blocked boolean NOT NULL DEFAULT false,
  usability_offline boolean NOT NULL DEFAULT false,
  usability_low_score boolean NOT NULL DEFAULT false,
  usability_redundant_ip boolean NOT NULL DEFAULT false,
  usability_gouging boolean NOT NULL DEFAULT false,
  usability_not_accepting_contracts boolean NOT NULL DEFAULT false,
  usability_not_announced boolean NOT NULL DEFAULT false,
  usability_not_completing_scan boolean NOT NULL DEFAULT false,

  score_age double precision NOT NULL,
  score_collateral double precision NOT NULL,
  score_interactions double precision NOT NULL,
  score_storage_remaining double precision NOT NULL,
  score_uptime double precision NOT NULL,
  score_version double precision NOT NULL,
  score_prices double precision NOT NULL,

  gouging_contract_err text,
  gouging_download_err text,
  gouging_gouging_err text,
  gouging_prune_err text,
  gouging_upload_err text,

  UNIQUE (db_autopilot_id, db_host_id),
  CONSTRAINT fk_host_checks_autopilot FOREIGN KEY (db_autopilot_id) REFERENCES autopilots (id) ON DELETE CASCADE,
  CONSTRAINT fk_host_checks_host FOREIGN KEY (db_host_id) REFERENCES hosts (id) ON DELETE CASCADE
);

CREATE INDEX idx_host_checks_usability_blocked ON host_checks (usability_blocked);
CREATE INDEX idx_host_checks_usability_offline ON host_checks (usability_offline);
CREATE INDEX idx_host_checks_usability_low_score ON host_checks (usability_low_score);
CREATE INDEX idx_host_checks_usability_redundant_ip ON host_checks (usability_redundant_ip);
CREATE INDEX idx_host_checks_usability_gouging ON host_checks (usability_gouging);
CREATE INDEX idx_host_checks_usability_not_accepting_contracts ON host_checks (usability_not_accepting_contracts);
CREATE INDEX idx_host_checks_usability_not_announced ON host_checks (usability_not_announced);
CREATE INDEX idx_host_checks_usability_not_completing_scan ON host_checks (usability_not_completing_scan);
CREATE INDEX idx_host_checks_score_age ON host_checks (score_age);
CREATE INDEX idx_host_checks_score_collateral ON host_checks (score_collateral);
CREATE INDEX idx_host_checks_score_interactions ON host_checks (score_interactions);
CREATE INDEX idx_host_checks_score_storage_remaining ON host_checks (score_storage_remaining);
CREATE INDEX idx_host_checks_score_uptime ON host_checks (score_uptime);
CREATE INDEX idx_host_checks_score_version ON host_checks (score_version);
CREATE INDEX idx_host_checks_score_prices ON host_checks (score_prices);


-- create default bucket
INSERT INTO buckets (created_at, name) VALUES (CURRENT_TIMESTAMP, 'default');
