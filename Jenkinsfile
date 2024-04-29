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
                    def sshCommand = 'pwd && ls -al'
                    sh sshCommand
                }
            }
        }
    }
    post {
        always {
            emailext body: '''
            Hi,
            $PROJECT_NAME - Build # $BUILD_NUMBER - $BUILD_STATUS:
            Check console output at $BUILD_URL to view the results.
            Thanks,
            Jenkins
            ''',
            subject: '$PROJECT_NAME - Build # $BUILD_NUMBER - $BUILD_STATUS:',
            to: 'himan.7525@gmail.com',
            replyTo: '$DEFAULT_REPLYTO'
        }
    }
}
