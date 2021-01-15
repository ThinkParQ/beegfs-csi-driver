pipeline {
    agent any

    options {
        timestamps()
        timeout(time: 1, unit: 'HOURS')
        buildDiscarder(logRotator(artifactNumToKeepStr: '15'))
    }

    stages {
        stage("Unit Test") {
            options {
                timeout(time: 5, unit: 'MINUTES')
            }
            steps {
                sh 'make test'
            }
        }
    }

    post {
        cleanup {
            deleteDir()
        }
    }
}