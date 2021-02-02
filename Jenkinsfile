// Copyright 2021 NetApp, Inc. All Rights Reserved.
// Licensed under the Apache License, Version 2.0.

projectVersion = '1.0'  // Increment this value when master branch refers to a different version.
if (env.BRANCH_NAME.matches('release-.+')) {
    projectVersion = env.BRANCH_NAME.split('-')[1]  // A release branch carries its own version.
}
paddedBuildNumber = env.BUILD_NUMBER.padLeft(4, '0')

def imageName = 'beegfs-csi-driver'  // release-tools gives significance to the name of the /cmd/beegfs-csi-driver directory.
def releaseToolsImageTag = 'beegfs-csi-driver:latest'  // The "make container" method in build.make uses this tag.

hubProjectName = 'esg-beegfs-csi-driver'
// Replace projectVersion with a custom version to do an experimental Black Duck scan.
hubProjectVersion = projectVersion
// Replace the below conditional with true to force a Black Duck scan. Don't do this unless you have also modified
// hubProjectVersion or you know exactly what you are doing.
shouldHubScan = env.BRANCH_NAME.matches('(master)|(release-.+)')

// We do NOT rely on release-tools tagging mechanism for internal builds because it does not provide mechanisms for
// overwriting image tags, etc.
imageTag = "docker.repo.eng.netapp.com/globalcicd/apheleia/${imageName}:${env.BRANCH_NAME}"  // e.g. .../globalcicd/apheleia/beegfsplugin:my-branch
uniqueImageTag = "${imageTag}-${paddedBuildNumber}"  // e.g. .../globalcicd/apheleia/beegfsplugin:my-branch-0005
if (env.BRANCH_NAME.matches('(master)|(release-.+)')) {
    imageTag = "docker.repo.eng.netapp.com/global/apheleia/${imageName}:v${projectVersion}"  // e.g. .../global/apheleia/beegfs-csi-driver:v1.0
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
                // (e.g. beegfs-csi-driver:latest). Make sure each node is only using this tag for one build at a time.
                lock(resource: "${releaseToolsImageTag}-${env.NODE_NAME}") {
                    sh """
                        set +e  # don't exit on failure
                        make container
                        RETURN_CODE=\$?  # remember return code
                        docker tag ${releaseToolsImageTag} ${uniqueImageTag}
                        docker rmi ${imageName}:latest  # clean up before releasing lock
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
                expression { shouldHubScan }
            }
            steps {
                // Do not scan the vendor directory. Everything in the vendor director is already discovered by the
                // GO_MOD detector and scanning it provides duplicate results with erroneous versions.
                synopsys_detect detectProperties: """
                    --detect.project.name=${hubProjectName} \
                    --detect.project.version.name=${hubProjectVersion} \
                    --detect.code.location.name=${hubProjectName}-${hubProjectVersion}-application \
                    --detect.project.code.location.unmap=true \
                    --detect.blackduck.signature.scanner.exclusion.name.patterns=vendor
                """
                synopsys_detect detectProperties: """
                    --detect.project.name=${hubProjectName} \
                    --detect.project.version.name=${hubProjectVersion} \
                    --detect.code.location.name=${hubProjectName}-${hubProjectVersion}-container \
                    --detect.project.code.location.unmap=true \
                    --detect.docker.image=${uniqueImageTag} \
                    --detect.docker.passthrough.service.distro.default=apk \
                    --detect.tools.excluded=DETECTOR
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
