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
                    def sshCommand = 'pwd && ls -al && cd /var/www/html/AutomationService && pwd && git pull origin main'
                    sh sshCommand
                }
            }
        }
    }
}
