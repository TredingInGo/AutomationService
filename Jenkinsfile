pipeline {
    agent any
    stages {
        stage('Checkout') {
            steps {
                git branch: 'main', credentialsId: 'go-server', url: 'https://github.com/TredingInGo/AutomationService.git'
            }
        }
        stage('Commands') {
            steps {
                script {
                    def mvCommand = 'pwd && ls -al && cp -R /var/lib/jenkins/workspace/go-pipeline/* /var/www/html/AutomationService && go build '
                    sh mvCommand
                }
            }
        }
    }
}
