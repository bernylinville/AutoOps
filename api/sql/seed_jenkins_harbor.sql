-- Seed data for Jenkins and Harbor credentials in config_account table
-- Run this after AutoOps database is initialized
-- NOTE: Passwords are AES-encrypted. Use the AutoOps admin UI to create real credentials.

-- Jenkins server (type = 4)
-- Host: 10.0.17.204, Port: 80 (Gateway forwards to 8080)
-- Default admin credentials from Jenkins Helm chart: admin/admin or configured in values.yaml
INSERT INTO config_account (alias, host, port, name, password, type, remark, created_at, updated_at)
VALUES (
    'pukka-jenkins',
    '10.0.17.204',
    80,
    'admin',
    'ENCRYPTED_PASSWORD_HERE',
    4,
    'Pukka GitOps Jenkins server',
    NOW(),
    NOW()
);

-- Harbor server (type = 5)
-- Host: 10.0.17.205, Port: 80
-- Default admin credentials from Harbor Helm chart: admin/Harbor12345
INSERT INTO config_account (alias, host, port, name, password, type, remark, created_at, updated_at)
VALUES (
    'pukka-harbor',
    '10.0.17.205',
    80,
    'admin',
    'ENCRYPTED_PASSWORD_HERE',
    5,
    'Pukka GitOps Harbor registry',
    NOW(),
    NOW()
);

-- Note: To get the encrypted password, use the AutoOps API or admin UI:
-- 1. Go to Configuration Center -> Account Auth
-- 2. Create new account with type Jenkins (4) or Harbor (5)
-- 3. The password will be automatically AES-encrypted
-- Or use the util.AESEncrypt() function in Go code

-- After inserting, note the IDs and update the ClusterTarget records:
-- UPDATE deploy_cluster_target SET jenkins_server_id = <jenkins_id>, harbor_server_id = <harbor_id> WHERE id = <target_id>;
