projectVersion = '1.0'  // Increment this value when master branch refers to a different version.
if (env.BRANCH_NAME.matches('release-.+')) {
    projectVersion = env.BRANCH_NAME.split('-')[1]  // A release branch carries its own version.
}
def paddedBuildNumber = env.BUILD_NUMBER.padLeft(4, '0')

def imageName = 'beegfsplugin'  // release-tools gives significance to the name of the /cmd/beegfsplugin directory.
def releaseToolsImageTag = 'beegfsplugin:latest'  // The "make container" method in build.make uses this tag.

def hubProjectName = 'esg-beegfs-csi-driver'
def hubProjectVersion = projectVersion

// We do NOT rely on release-tools tagging mechanism for internal builds because it does not provide mechanisms for
// overwriting image tags, etc.
def uniqueImageTag = "docker.repo.eng.netapp.com/globalcicd/apheleia/${imageName}:${env.BRANCH_NAME}-${paddedBuildNumber}"  // e.g. .../globalcicd/apheleia/beegfsplugin:my-branch-0005
def imageTag = "docker.repo.eng.netapp.com/globalcicd/apheleia/${imageName}:${env.BRANCH_NAME}"  // e.g. .../globalcicd/apheleia/beegfsplugin:my-branch
if (env.BRANCH_NAME.matches('(master)|(release-.+)')) {
    imageTag = "docker.repo.eng.netapp.com/global/apheleia/${imageName}:v${projectVersion}"  // e.g. .../global/apheleia/beegfsplugin:v1.0
}

pipeline {
    agent any

    options {
        timestamps()
        timeout(time: 1, unit: 'HOURS')
        buildDiscarder(logRotator(artifactNumToKeepStr: '15'))
    }

    stages {
        stage('Unit Test') {
            options {
                timeout(time: 5, unit: 'MINUTES')
            }
            steps {
                // release-tools always uses a container named k8s-shellcheck in its test. Make sure each node is only
                // using this tag for one build at a time.
                lock(resource: "k8s-shellcheck-${env.NODE_NAME}") {
                    sh 'make test'
                }
            }
        }
        stage('Build Container') {
            steps {
                // release-tools always builds the container with the same releaseToolsImageTag
                // (e.g. beegfsplugin:latest). Make sure each node is only using this tag for one build at a time.
                lock(resource: "${releaseToolsImageTag}-${env.NODE_NAME}") {
                    sh """
                        set +e  # don't exit on failure
                        make container
                        RETURN_CODE=\$?  # remember return code
                        docker tag ${releaseToolsImageTag} ${uniqueImageTag}
                        docker rmi beegfsplugin:latest  # clean up before releasing lock
                        exit \$RETURN_CODE
                    """
                }
            }
        }
        stage('Push Container') {
            options {
                timeout(time: 5, unit: 'MINUTES')
            }
            steps {
                withDockerRegistry([credentialsId: 'mswbuild', url: 'https://docker.repo.eng.netapp.com']) {
                    sh """
                        docker tag ${uniqueImageTag} ${imageTag}
                        docker push ${uniqueImageTag}
                        docker push ${imageTag}
                    """
                }
            }
        }
        stage("BlackDuck Scan") {
            options {
                timeout(time: 10, unit: 'MINUTES')
            }
            when {
                // Only scan master and release branches.
                expression { env.BRANCH_NAME.matches('(master)|(release-.+)') }
            }
            steps {
                synopsys_detect detectProperties: """
                    --detect.project.name=${hubProjectName} \
                    --detect.project.version.name=${hubProjectVersion} \
                    --detect.code.location.name=${hubProjectName}-${hubProjectVersion} \
                    --detect.project.code.location.unmap=true \
                    --detect.docker.image=${uniqueImageTag} \
                    --detect.docker.passthrough.service.distro.default=apk
                """
            }
        }
    }

    post {
        cleanup {
            sh "docker rmi ${imageTag} ${uniqueImageTag} || true"
            deleteDir()
        }
    }
}