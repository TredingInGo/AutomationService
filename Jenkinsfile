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
                    // Set PATH environment variable to include Go binary directory
                    env.PATH = "${tool 'GO'}/bin:${env.PATH}"
                    def mvCommand = 'pwd && ls -al && cp -R /var/lib/jenkins/workspace/go-pipeline/* /var/www/html/AutomationService && cd /var/www/html/AutomationService && go build && systemctl restart ngix.service'
                    sh mvCommand
                }
            }
        }
    }
}
