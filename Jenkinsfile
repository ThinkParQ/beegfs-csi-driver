// Copyright 2021 NetApp, Inc. All Rights Reserved.
// Licensed under the Apache License, Version 2.0.

// Set up build parameters so any branch can be manually rebuilt with different values.
properties([
    parameters ([
        string(name: 'hubProjectVersion', defaultValue: '', description: 'Set this to force a Black Duck scan and ' +
               'manually tag it to a particular Black Duck version (e.g. 1.0.1).')
    ])
])

paddedBuildNumber = env.BUILD_NUMBER.padLeft(4, '0')
imageName = 'beegfs-csi-driver'  // release-tools gives significance to the name of the /cmd/beegfs-csi-driver directory.
releaseToolsImageTag = 'beegfs-csi-driver:latest'  // The "make container" method in build.make uses this tag.

hubProjectName = 'esg-beegfs-csi-driver'
hubProjectVersion = ''
shouldHubScan = false
if (params.hubProjectVersion != '') {
    // Scan the project and tag the manually selected version if the hubProjectVersion build parameter is set.
    hubProjectVersion = params.hubProjectVersion
    shouldHubScan = true
} else if (env.BRANCH_NAME.matches('(master)|(release-.+)')) {
    // Scan the project and tag the branch name if this is the release or master branch.
    hubProjectVersion = env.BRANCH_NAME
    shouldHubScan = true
}

// We do NOT rely on release-tools tagging mechanism for internal builds because it does not provide mechanisms for
// overwriting image tags, etc.
imageTag = "docker.repo.eng.netapp.com/globalcicd/apheleia/${imageName}:${env.BRANCH_NAME}"  // e.g. .../globalcicd/apheleia/beegfsplugin:my-branch
uniqueImageTag = "${imageTag}-${paddedBuildNumber}"  // e.g. .../globalcicd/apheleia/beegfsplugin:my-branch-0005

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
                    script {
                        if (env.BRANCH_NAME.matches('(master)|(release-.+)|(PR-.+)')) {
                            // When JOB_NAME is empty, the conditional logic in release-tools/verify-vendor.sh allows
                            // for vendor testing.
                            sh 'JOB_NAME= make test'
                        } else {
                            // When JOB_NAME is not empty (automatically set by Jenkins), the conditional logic in
                            // release-tools/verify-vendor.sh does not allow for vendor testing. This is good, because
                            // vendor testing forces a download of all modules, which is time/bandwidth intensive.
                            sh 'make test'
                        }
                    }
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
                    --detect.cleanup=false \
                    --detect.output.path=blackduck \
                    --detect.project.code.location.unmap=true \
                    --detect.detector.search.depth=50 \
                    --detect.code.location.name=${hubProjectName}_${hubProjectVersion}_application_code \
                    --detect.bom.aggregate.name=${hubProjectName}_${hubProjectVersion}_application_bom \
                    --detect.detector.search.exclusion.paths=vendor/,blackduck/ \
                    --detect.blackduck.signature.scanner.exclusion.patterns=/vendor/,/blackduck/
                """
                synopsys_detect detectProperties: """
                    --detect.project.name=${hubProjectName} \
                    --detect.project.version.name=${hubProjectVersion} \
                    --detect.cleanup=false \
                    --detect.output.path=blackduck \
                    --detect.project.code.location.unmap=true \
                    --detect.detector.search.depth=50 \
                    --detect.code.location.name=${hubProjectName}_${hubProjectVersion}_container_code \
                    --detect.bom.aggregate.name=${hubProjectName}_${hubProjectVersion}_container_bom \
                    --detect.docker.image=${uniqueImageTag} \
                    --detect.docker.passthrough.service.distro.default=apk \
                    --detect.docker.path=/usr/bin/docker \
                    --detect.tools=DOCKER \
                    --detect.tools=SIGNATURE_SCAN
                """
            }
            post {
                success {
                    // Exclude .tar.gz files to avoid archiving large(ish) Docker extractions.
                    archiveArtifacts(artifacts: 'blackduck/runs/**', excludes: '**/*.tar.gz')
                }
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
