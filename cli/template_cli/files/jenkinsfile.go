package files

import "fmt"

func (g *FileGenerator) generateJenkinsfile() string {
	serviceName := g.Info.ServiceName
	if serviceName == "" {
		serviceName = "app"
	}
	imageName := g.Info.ImageName
	if imageName == "" {
		imageName = serviceName
	}

	return fmt.Sprintf(`pipeline {
    agent any
    
    environment {
        APP_NAME = '%s'
        DOCKER_IMAGE = '%s'
        DOCKER_REGISTRY = 'your-registry.com'
        GO_VERSION = '1.21'
    }
    
    stages {
        stage('Checkout') {
            steps {
                checkout scm
            }
        }
        
        stage('Setup Go') {
            steps {
                sh '''
                    go version
                    go mod download
                '''
            }
        }
        
        stage('Lint') {
            steps {
                sh '''
                    go vet ./...
                '''
            }
        }
        
        stage('Test') {
            steps {
                sh '''
                    go test -v -race -coverprofile=coverage.out ./...
                    go tool cover -html=coverage.out -o coverage.html
                '''
            }
            post {
                always {
                    archiveArtifacts artifacts: 'coverage.html', allowEmptyArchive: true
                }
            }
        }
        
        stage('Build') {
            steps {
                sh '''
                    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
                        -ldflags='-w -s' \
                        -o ${APP_NAME} .
                '''
            }
        }
        
        stage('Docker Build') {
            when {
                branch 'main'
            }
            steps {
                script {
                    def imageTag = "${DOCKER_REGISTRY}/${DOCKER_IMAGE}:${BUILD_NUMBER}"
                    sh "docker build -t ${imageTag} ."
                    sh "docker tag ${imageTag} ${DOCKER_REGISTRY}/${DOCKER_IMAGE}:latest"
                }
            }
        }
        
        stage('Docker Push') {
            when {
                branch 'main'
            }
            steps {
                withCredentials([usernamePassword(
                    credentialsId: 'docker-registry-credentials',
                    usernameVariable: 'DOCKER_USER',
                    passwordVariable: 'DOCKER_PASS'
                )]) {
                    sh '''
                        echo $DOCKER_PASS | docker login $DOCKER_REGISTRY -u $DOCKER_USER --password-stdin
                        docker push ${DOCKER_REGISTRY}/${DOCKER_IMAGE}:${BUILD_NUMBER}
                        docker push ${DOCKER_REGISTRY}/${DOCKER_IMAGE}:latest
                    '''
                }
            }
        }
        
        stage('Deploy') {
            when {
                branch 'main'
            }
            steps {
                echo 'Deploying to production...'
            }
        }
    }
    
    post {
        always {
            cleanWs()
        }
        success {
            echo 'Pipeline completed successfully!'
        }
        failure {
            echo 'Pipeline failed!'
        }
    }
}
`, serviceName, imageName)
}
