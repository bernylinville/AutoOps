# End-to-End Pipeline Test Configuration

## Infrastructure Setup

### Jenkins (10.0.17.204)
- **URL**: http://10.0.17.204/
- **Namespace**: jenkins
- **Service**: jenkins:8080
- **Gateway**: jenkins-gateway (MetalLB IP 10.0.17.204)

### Harbor (10.0.17.205)
- **URL**: http://10.0.17.205/
- **Namespace**: harbor
- **Services**: harbor-core:80, harbor-portal:80
- **Gateway**: harbor-gateway (MetalLB IP 10.0.17.205)

### GitLab
- **URL**: http://gayhub.seeingtv.com
- **Repo**: http://gayhub.seeingtv.com/ipaas/pukka-gitops.git

## AutoOps Configuration Steps

### 1. Add Jenkins Credentials to config_account

```sql
INSERT INTO config_account (alias, host, port, name, password, type, remark, created_at, updated_at)
VALUES (
    'pukka-jenkins',
    '10.0.17.204',
    80,
    'admin',
    'ENCRYPTED_PASSWORD',
    4,  -- JenkinsAccountType
    'Pukka GitOps Jenkins',
    NOW(),
    NOW()
);
```

Note: Get the actual admin password from Jenkins. The password must be AES-encrypted using AutoOps' encryption.

### 2. Add Harbor Credentials to config_account

```sql
INSERT INTO config_account (alias, host, port, name, password, type, remark, created_at, updated_at)
VALUES (
    'pukka-harbor',
    '10.0.17.205',
    80,
    'admin',
    'ENCRYPTED_PASSWORD',
    5,  -- HarborAccountType
    'Pukka GitOps Harbor',
    NOW(),
    NOW()
);
```

Note: Default Harbor admin password is usually `Harbor12345` unless changed.

### 3. Create Application with JenkinsEnv

```sql
-- Create application for Spring PetClinic
INSERT INTO app_application (name, description, status, created_at, updated_at)
VALUES ('spring-petclinic', 'Spring PetClinic Demo App', 1, NOW(), NOW());

-- Note the application ID, then create JenkinsEnv mapping
INSERT INTO app_jenkins_env (app_id, env_name, jenkins_server_id, job_name, created_at, updated_at)
VALUES (
    <application_id>,
    'test',
    <jenkins_server_id>,  -- from config_account
    'spring-petclinic-build',
    NOW(),
    NOW()
);
```

### 4. Update ClusterTarget with Jenkins/Harbor IDs

```sql
UPDATE deploy_cluster_target 
SET jenkins_server_id = <jenkins_id>,
    harbor_server_id = <harbor_id>
WHERE id = <target_id>;
```

### 5. Create Jenkins Job for Spring PetClinic

Create a Jenkins Pipeline job named `spring-petclinic-build` with the following pipeline script:

```groovy
pipeline {
    agent any
    
    parameters {
        string(name: 'GIT_REF', defaultValue: 'main', description: 'Git branch or tag')
    }
    
    environment {
        HARBOR_REGISTRY = '10.0.17.205'
        HARBOR_PROJECT = 'library'
        HARBOR_REPO = 'spring-petclinic'
        IMAGE_TAG = "${BUILD_NUMBER}"
    }
    
    stages {
        stage('Checkout') {
            steps {
                git branch: "${params.GIT_REF}", 
                    url: 'http://gayhub.seeingtv.com/ipaas/spring-petclinic.git'
            }
        }
        
        stage('Build') {
            steps {
                sh './mvnw clean package -DskipTests'
            }
        }
        
        stage('Build Image') {
            steps {
                script {
                    def image = docker.build("${HARBOR_REGISTRY}/${HARBOR_PROJECT}/${HARBOR_REPO}:${IMAGE_TAG}")
                }
            }
        }
        
        stage('Push to Harbor') {
            steps {
                script {
                    docker.withRegistry("http://${HARBOR_REGISTRY}", 'harbor-credentials') {
                        docker.image("${HARBOR_REGISTRY}/${HARBOR_PROJECT}/${HARBOR_REPO}:${IMAGE_TAG}").push()
                    }
                }
                // Output the image tag for AutoOps to extract
                echo "IMAGE_TAG=${IMAGE_TAG}"
            }
        }
    }
    
    post {
        always {
            echo "Build completed with tag: ${IMAGE_TAG}"
        }
    }
}
```

### 6. Create Harbor Project

Login to Harbor UI (http://10.0.17.205/) and create a project named `library` (or use existing `library` project).

### 7. Clone Spring PetClinic to GitLab

```bash
git clone https://github.com/spring-projects/spring-petclinic.git
cd spring-petclinic
git remote add gitlab http://gayhub.seeingtv.com/ipaas/spring-petclinic.git
git push gitlab main
```

### 8. Test the Pipeline

Send an Agent deploy request with:

```json
{
  "requesterExternalType": "dingtalk",
  "requesterExternalId": "user_id",
  "mode": "gitops",
  "workflowKind": "build_deploy",
  "applicationId": <app_id>,
  "resourceType": "deployment",
  "clusterTargetId": <target_id>,
  "releaseName": "spring-petclinic",
  "namespace": "default",
  "replicas": 1,
  "reason": "Test pipeline",
  "gitRef": "main",
  "harborProject": "library",
  "harborRepository": "spring-petclinic",
  "artifactTag": "latest"
}
```

## Expected Flow

1. Agent request created with `workflow_kind=build_deploy`
2. DeployRequest status = `pending_approval`
3. PipelineRun created with status = `pending`
4. Approval triggered via DingTalk
5. After approval:
   - DeployRequest approval_status = `approved`
   - PipelineScheduler picks up PipelineRun
   - Claim updates status to `building`
6. Build stage:
   - Trigger Jenkins job `spring-petclinic-build` with GIT_REF=main
   - Poll until build completes
   - Extract IMAGE_TAG from build log
7. Scan stage:
   - Get Harbor artifact with extracted tag
   - Trigger vulnerability scan
   - Poll until scan completes
   - Evaluate scan policy (default: 0 critical, 0 high)
8. Deploy stage:
   - Update DeployRequest image with final image ref
   - Call AutoExecuteApprovedDeployRequest
   - Execute GitOps deployment
9. Notify stage:
   - Send result notification via DingTalk

## Troubleshooting

### PipelineRun not picked up
- Check DeployRequest approval_status = `approved`
- Check PipelineRun status = `pending`
- Check PipelineScheduler logs

### Jenkins build fails
- Check Jenkins credentials in config_account
- Verify Jenkins job exists and is buildable
- Check Jenkins build logs

### Harbor scan fails
- Check Harbor credentials in config_account
- Verify artifact exists in Harbor
- Check Harbor scan status

### Deploy fails
- Check ClusterTarget kubeconfig
- Verify GitOps repo access
- Check ArgoCD sync status
