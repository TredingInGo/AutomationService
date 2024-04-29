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
                    def sshCommand = 'pwd && ls -al && sudo su && mv /var/lib/jenkins/workspace/go-pipeline/* /var/www/html/AutomationService '
                    sh sshCommand
                }
            }
        }
    }
}
