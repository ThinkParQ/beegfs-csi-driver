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
remoteImageName = "docker.repo.eng.netapp.com/globalcicd/apheleia/${imageName}"
imageTag = "${remoteImageName}:${env.BRANCH_NAME}"  // e.g. .../globalcicd/apheleia/beegfs-csi-driver:my-branch
uniqueImageTag = "${imageTag}-${paddedBuildNumber}"  // e.g. .../globalcicd/apheleia/beegfs-csi-driver:my-branch-0005

operatorImageName = "${remoteImageName}-operator"
operatorImageTag = "${operatorImageName}:${env.BRANCH_NAME}"
uniqueOperatorImageTag = "${operatorImageTag}-${paddedBuildNumber}"

bundleImageName = "${operatorImageName}-bundle"
bundleImageTag = "${bundleImageName}:${env.BRANCH_NAME}"
uniqueBundleImageTag = "${bundleImageTag}-${paddedBuildNumber}"

pipeline {
    agent any

    options {
        timestamps()
        timeout(time: 3, unit: 'HOURS')
        buildDiscarder(logRotator(artifactNumToKeepStr: '15'))
    }

    stages {
        stage('Unit Test') {
            options {
                timeout(time: 10, unit: 'MINUTES')
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
        // The operator built in this step can be retagged and released to Docker Hub as needed.
        stage('Build and Push Operator') {
            options {
                timeout(time: 5, unit: 'MINUTES')
            }
            steps {
                withDockerRegistry([credentialsId: 'mswbuild', url: 'https://docker.repo.eng.netapp.com']) {
                    sh """
                        cd operator
                        make -e ENVTEST_ASSETS_DIR=/var/lib/jenkins/operator-sdk-envtest -e IMG=${uniqueOperatorImageTag} build docker-build
                        # Build bundle without modification to verify that generated code and manifests are up to date.
                        make -e ENVTEST_ASSETS_DIR=/var/lib/jenkins/operator-sdk-envtest bundle
                        if [[ \$(git diff) ]]
                        then
                            # The above make steps have run all generators. The developer making changes should also 
                            # have run all generators and committed the result. Do not proceed if the generators run 
                            # here produce different output than the developer committed.
                            echo "ERROR: Generated code and/or manifests are not up to date"
                            git diff
                            exit 1
                        fi
                        docker tag ${uniqueOperatorImageTag} ${operatorImageTag}
                        make -e IMG=${uniqueOperatorImageTag} docker-push
                        make -e IMG=${operatorImageTag} docker-push
                    """
                }
            }
        }
        // The bundle container built in this step can only be used for testing, as it references an operator image tag
        // that does not exist on Docker Hub. This is fine because a bundle container is not actually used to release
        // an operator (the pristine bundle directory is used instead).
        stage('Build and Push Bundle') {
            options {
                timeout(time: 5, unit: 'MINUTES')
            }
            steps {
                withDockerRegistry([credentialsId: 'mswbuild', url: 'https://docker.repo.eng.netapp.com']) {
                    sh """
                        cd operator
                        make -e ENVTEST_ASSETS_DIR=/var/lib/jenkins/operator-sdk-envtest -e IMG=${uniqueOperatorImageTag} -e BUNDLE_IMG=${uniqueBundleImageTag} bundle bundle-build bundle-push
                        make -e ENVTEST_ASSETS_DIR=/var/lib/jenkins/operator-sdk-envtest -e IMG=${operatorImageTag} -e BUNDLE_IMG=${bundleImageTag} bundle bundle-build bundle-push
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
            environment {
                // TODO(webere, A251): Stop pinning this version when the issue with skipping directories in >v7.0.0 is
                // resolved.
                DETECT_LATEST_RELEASE_VERSION = '6.9.1'
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
                synopsys_detect detectProperties: """
                    --detect.project.name=${hubProjectName} \
                    --detect.project.version.name=${hubProjectVersion} \
                    --detect.cleanup=false \
                    --detect.output.path=blackduck \
                    --detect.project.code.location.unmap=true \
                    --detect.detector.search.depth=50 \
                    --detect.code.location.name=${hubProjectName}_${hubProjectVersion}_operator_container_code \
                    --detect.bom.aggregate.name=${hubProjectName}_${hubProjectVersion}_operator_container_bom \
                    --detect.docker.image=${uniqueOperatorImageTag} \
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
        stage("Integration Testing") {
            when {
                expression {
                    return env.BRANCH_NAME.startsWith('PR-') || env.BRANCH_NAME.matches('master') ||
                        !currentBuild.getBuildCauses('hudson.model.Cause$UserIdCause').isEmpty()
                }
            }
            options {
                timeout(time: 6, unit: 'HOURS')
            }
            environment {
                // The disruptive test suite will try to SSH into k8s cluster nodes, defaulting as the jenkins user,
                // which doesn't exist on those nodes. This changes the test suite to SSH as the root user instead
                KUBE_SSH_USER = "root"
            }
            steps {
                script {
                    if (env.BRANCH_NAME.startsWith('PR-') && currentBuild.getBuildCauses('hudson.model.Cause$UserIdCause').isEmpty()) {
                        // We want to ensure the integration tests pass before merging pull requests, but we don't want them to run
                        // after every push to conserve resources. This stage will only pass on a PR if it is triggered manually
                        // in Jenkins and all integration tests pass.
                        error("Integration tests are not run automatically by Bitbucket. Trigger a build manually to run integration tests.")
                    }

                    String[][] testEnvironments
                    if (env.BRANCH_NAME.matches('master')) {
                        testEnvironments = [
                            // Each cluster must use a different staticVolDirName to avoid collisions.
                            ["1.18", "beegfs-7.1.5", "prod-1.18", "static1"],
                            ["1.18", "beegfs-7.2", "prod-1.18", "static1"],
                            ["1.19-rdma", "beegfs-7.2-rdma", "prod", "static2"],
                            ["1.20", "beegfs-7.1.5", "prod", "static3"],
                            ["1.20", "beegfs-7.2", "prod", "static3"]
                        ]
                    }
                    else {
                        testEnvironments = [
                            // Each cluster must use a different staticVolDirName to avoid collisions.
                            ["1.18", "beegfs-7.2", "prod-1.18", "static1"],
                            ["1.19-rdma", "beegfs-7.2-rdma", "prod", "static2"],
                            ["1.20", "beegfs-7.2", "prod", "static3"]
                        ]
                    }

                    def integrationJobs = [:]
                    testEnvironments.each { k8sCluster, beegfsHost, deployDir, staticVolDirName ->
                        integrationJobs["kubernetes: ${k8sCluster}, beegfs: ${beegfsHost}"] = {
                            runIntegrationSuite(k8sCluster, beegfsHost, deployDir, staticVolDirName)
                        }
                    }
                    parallel integrationJobs
                }
            }
            post {
                always {
                    junit "test/e2e/junit/*.xml"
                }
            }
        }
    }

    post {
        cleanup {
            sh """
                docker image list | grep ${env.BRANCH_NAME} | awk '{ print \$1 ":" \$2 }' | xargs -r docker rmi
            """
            deleteDir()
        }
    }
}

def runIntegrationSuite(k8sCluster, beegfsHost, deployDir, staticVolDirName) {
    // Always skip the broken subpath test.
    String ginkgoSkipRegex = "should be able to unmount after the subpath directory is deleted"
    // Skip the [Slow] tests except on master.
    // Ginkgo requires a \ escape and Groovy requires a \ escape for every \.
    if (!env.BRANCH_NAME.matches('master')) {
        ginkgoSkipRegex += "|\\[Slow\\]"
    }

    def jobID = "${k8sCluster}-${beegfsHost}"
    // Per documentation, always make kustomizations in deploy/k8s/prod.
    sh """
        cp -r deploy/ deploy-${jobID}/
        (cd deploy-${jobID}/k8s/prod && ${HOME}/kustomize edit set image beegfs-csi-driver=${remoteImageName}:${env.BRANCH_NAME})
    """
    lock(resource: "${k8sCluster}") {
        // Credentials variables are always local to the withCredentials block, so multiple
        // instances of the KUBECONFIG variable can exist without issue when running in parallel
        withCredentials([file(credentialsId: "kubeconfig-${k8sCluster}", variable: "KUBECONFIG")]) {
            String clusterGinkgoSkipRegex = ginkgoSkipRegex
            if (k8sCluster.contains("1.18")) {
                // Generic ephemeral volumes aren't supported in v1.18, but the end-to-end tests
                // incorrectly identify our v1.18 cluster as being ephemeral-capable.
                clusterGinkgoSkipRegex += "|ephemeral"
            }

            try {
                // The two kubectl get ... lines are used to clean up any beegfs CSI driver currently
                // running on the cluster. We can't simply delete using -k deploy/k8s/prod/ because a previous
                // user might have deployed the driver using a different deployment scheme.
                sh """
                    echo 'Running test using kubernetes version ${k8sCluster} with beegfs version ${beegfsHost}'
                    kubectl get sts -A | grep csi-beegfs | awk '{print \$2 " -n " \$1}' | xargs kubectl delete --cascade=foreground sts || true
                    kubectl get ds -A | grep csi-beegfs | awk '{print \$2 " -n " \$1}' | xargs kubectl delete --cascade=foreground ds || true
                    cp test/env/${beegfsHost}/csi-beegfs-config.yaml deploy-${jobID}/k8s/prod/csi-beegfs-config.yaml
                    cp test/env/${beegfsHost}/csi-beegfs-connauth.yaml deploy-${jobID}/k8s/prod/csi-beegfs-connauth.yaml
                    kubectl apply -k deploy-${jobID}/k8s/${deployDir}/
                    ginkgo -v -p -nodes 8 -noColor -skip '${clusterGinkgoSkipRegex}|\\[Disruptive\\]|\\[Serial\\]' -timeout 60m ./test/e2e/ -- -report-dir ./junit -report-prefix parallel-${jobID} -static-vol-dir-name ${staticVolDirName}
                    ginkgo -v -noColor -skip '${clusterGinkgoSkipRegex}' -focus '\\[Disruptive\\]|\\[Serial\\]' -timeout 60m ./test/e2e/ -- -report-dir ./junit -report-prefix serial-${jobID} -static-vol-dir-name ${staticVolDirName}
                    kubectl delete --cascade=foreground -k deploy-${jobID}/k8s/${deployDir}/
                """
            } catch (err) {
                sh "kubectl delete --cascade=foreground -k deploy-${jobID}/k8s/${deployDir}/ || true"
                throw err
            }
        }
    }
}
