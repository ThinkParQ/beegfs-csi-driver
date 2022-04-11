// Copyright 2021 NetApp, Inc. All Rights Reserved.
// Licensed under the Apache License, Version 2.0.

// Set up build parameters so any branch can be manually rebuilt with different values.
properties([
    parameters([
        string(name: 'hubProjectVersion', defaultValue: '', description: 'Set this to force a Black Duck scan and ' +
            'manually tag it to a particular Black Duck version (e.g. 1.0.1).'),
        booleanParam(name: 'shouldEndToEndTest', defaultValue: false, description: 'Set this to true to force ' +
            'end-to-end testing for a branch build. Note that end-to-end testing always occurs on PR and master ' +
            'builds.')
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

shouldEndToEndTest = params.shouldEndToEndTest  // Definitely end-to-end test if requested.
if (env.BRANCH_NAME.matches('(master)|(release-.+)|PR-.+')) {  // Always end-to-end test master, release, and PR builds.
    shouldEndToEndTest = true
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
                        def testCommand = 'ACK_GINKGO_DEPRECATIONS=1.16.5 TESTARGS="-v -ginkgo.v" make test > ' +
                            'results/unit-test.log'
                        if (env.BRANCH_NAME.matches('(master)|(release-.+)|(PR-.+)')) {
                            // When JOB_NAME is empty, the conditional logic in release-tools/verify-vendor.sh allows
                            // for vendor testing.
                            sh "mkdir results/ && JOB_NAME= ${testCommand}"
                        } else {
                            // When JOB_NAME is not empty (automatically set by Jenkins), the conditional logic in
                            // release-tools/verify-vendor.sh does not allow for vendor testing. This is good, because
                            // vendor testing forces a download of all modules, which is time/bandwidth intensive.
                            sh "mkdir results/ && ${testCommand}"
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
        stage('Build, Test, and Push Operator') {
            options {
                timeout(time: 5, unit: 'MINUTES')
            }
            steps {
                // envtest sets up a variety of services that listen on different ports. While we can change the ports
                // used relatively easily, we cannot easily make the ports random. Better to make sure envtest is only
                // in use by one build at a time on a particular node.
                lock(resource: "envtest-${env.NODE_NAME}") {
                    sh """
                        cd operator
                        make -e IMG=${uniqueOperatorImageTag} build docker-build
                        # Build bundle without modification to verify that generated code and manifests are up to date.
                        make bundle
                        if [[ \$(git diff) ]]
                        then
                            # The above make steps have run all generators. The developer making changes should also 
                            # have run all generators and committed the result. Do not proceed if the generators run 
                            # here produce different output than the developer committed.
                            echo "ERROR: Generated code and/or manifests are not up to date"
                            git diff
                            exit 1
                        fi
                    """
                }
                withDockerRegistry([credentialsId: 'mswbuild', url: 'https://docker.repo.eng.netapp.com']) {
                    sh """
                        cd operator
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
                        make -e IMG=${uniqueOperatorImageTag} -e BUNDLE_IMG=${uniqueBundleImageTag} bundle bundle-build bundle-push
                        make -e IMG=${operatorImageTag} -e BUNDLE_IMG=${bundleImageTag} bundle bundle-build bundle-push
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
                    --detect.excluded.directories=vendor/,blackduck/ \
                    --detect.go.mod.enable.verification=true
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
        stage("End-to-End Test") {
            when {
                expression { shouldEndToEndTest }
            }
            options {
                timeout(time: 6, unit: 'HOURS')
            }
            environment {
                // OpenShift will not remember authorized keys between upgrades, so it is not practical to do an 
                // ssh-copy-id from a Jenkins worker node to each OpenShift node.
                SSH_OPENSHIFT = credentials('ssh-openshift')
            }
            steps {
                script {
                    TestEnvironment[] testEnvironments
                    // Most Pre-provisioned PV tests use createVolumeResource(), which calls createPVCPV(). createPVCPV()
                    // creates a PVC and PV that reference a Storage Class with the same name as the test Namespace, but it
                    // doesn't actually create this Storage Class. The OpenShift Pod admission webhook refuses to allow
                    // Pods that reference a non-existent storage class. For now, pass staticVolDirName=""
                    // TODO(webere, A289): Figure out a low-touch way to enable these tests in OpenShift.
                    if (env.BRANCH_NAME.matches('master')) {
                        testEnvironments = [
                            // Each cluster must use a different staticVolDirName to avoid collisions.
                            new TestEnvironment("1.20", "beegfs-7.1.5", "1.20", "static3", "root", false),
                            new TestEnvironment("1.20", "beegfs-7.2", "1.20", "static3", "root", false),
                            new TestEnvironment("1.21", "beegfs-7.1.5", "1.21", "static4", "root", false),
                            new TestEnvironment("1.21", "beegfs-7.2", "1.21", "static4", "root", false),
                            new TestEnvironment("1.22", "beegfs-7.1.5", "1.22", "static5", "root", false),
                            new TestEnvironment("1.22", "beegfs-7.2", "1.22", "static5", "root", false),
                            new TestEnvironment("1.23-ubuntu-rdma", "beegfs-7.2-rh8-rdma", "1.23", "static6", "user", false),
                            new TestEnvironment("1.23-ubuntu-rdma", "beegfs-7.1.5", "1.23", "static6", "user", false),
                            new TestEnvironment("openshift", "beegfs-7.1.5", "1.22", "", "root", true),
                            new TestEnvironment("openshift", "beegfs-7.2", "1.22", "", "root", true)
                        ]
                    } else {
                        testEnvironments = [
                            // Each cluster must use a different staticVolDirName to avoid collisions.
                            new TestEnvironment("1.20", "beegfs-7.2", "1.20", "static3", "root", false),
                            new TestEnvironment("1.21", "beegfs-7.2", "1.21", "static4", "root", false),
                            new TestEnvironment("1.22", "beegfs-7.2", "1.22", "static5", "root", false),
                            new TestEnvironment("1.23-ubuntu-rdma", "beegfs-7.2-rh8-rdma", "1.23", "static6", "user", false),
                            new TestEnvironment("openshift", "beegfs-7.2", "1.23", "", "root", true)
                        ]
                    }

                    def integrationJobs = [:]
                    testEnvironments.each { testEnv ->
                        integrationJobs["kubernetes: ${testEnv.k8sCluster}, beegfs: ${testEnv.beegfsHost}"] = {
                            runIntegrationSuite(testEnv)
                        }
                    }
                    parallel integrationJobs
                }
            }
        }
    }

    post {
        cleanup {
            archiveArtifacts(artifacts: 'results/**/*')
            sh """
                docker image list | grep ${env.BRANCH_NAME} | awk '{ print \$1 ":" \$2 }' | xargs -r docker rmi
            """
            deleteDir()
        }
    }
}

def runIntegrationSuite(TestEnvironment testEnv) {
    // Always skip the broken subpath test.
    String ginkgoSkipRegex = "should be able to unmount after the subpath directory is deleted"
    // Skip the [Slow] tests except on master.
    // Ginkgo requires a \ escape and Groovy requires a \ escape for every \.
    if (!env.BRANCH_NAME.matches('master')) {
        ginkgoSkipRegex += "|\\[Slow\\]"
    }

    def jobID = "${testEnv.k8sCluster}-${testEnv.beegfsHost}"
    def resultsDir = "results/${jobID}"
    sh "mkdir -p ${resultsDir}"
    def testCommand = "ginkgo -v -p -nodes 8 -noColor -skip '${ginkgoSkipRegex}|\\[Disruptive\\]|\\[Serial\\]' -timeout 60m ./test/e2e/ -- -report-dir ../../${resultsDir} -report-prefix parallel"
    def testCommandDisruptive = "ginkgo -v -noColor -skip '${ginkgoSkipRegex}' -focus '\\[Disruptive\\]|\\[Serial\\]' -timeout 60m ./test/e2e/ -- -report-dir ../../${resultsDir} -report-prefix serial"
    if (testEnv.staticVolDirName) {
        testCommand += " -static-vol-dir-name ${testEnv.staticVolDirName}"
        testCommandDisruptive += " -static-vol-dir-name ${testEnv.staticVolDirName}"
    }
    // Redirect output for easier reading.
    testCommand += " > ${resultsDir}/ginkgo-parallel.log 2>&1"
    testCommandDisruptive += " > ${resultsDir}/ginkgo-serial.log 2>&1"

    echo "Running test using kubernetes version ${testEnv.k8sCluster} with beegfs version ${testEnv.beegfsHost}"
    lock(resource: "${testEnv.k8sCluster}") {
        if (testEnv.useOperator) {
            withCredentials([
                usernamePassword(credentialsId: "credentials-${testEnv.k8sCluster}", usernameVariable: "OC_USERNAME", passwordVariable: "OC_PASSWORD"),
                string(credentialsId: "address-${testEnv.k8sCluster}", variable: "OC_ADDRESS")]) {
                try {
                    // We escape the $ on OC_ADDRESS, etc. to avoid Groovy interpolation for secrets.
                    //
                    // We are not using a secret KUBECONFIG here as we do in the non-OpenShift deployment. However,
                    // we still need to set KUBECONFIG to point to an empty file in the workspace so "oc login" won't
                    // modify the Jenkins user's ~/.kube/config.
                    //
                    // It's a bit awkward to include Scorecard testing here, because Scorecard currently only evaluates
                    // our bundle (doesn't really do integration tests) and currently isn't expected to get different
                    // results on different clusters. However:
                    //   1) We need kubeconfig access to a cluster to run Scorecard.
                    //   2) We may eventually write custom tests with cluster-dependent results.
                    sh """                   
                        export KUBECONFIG="${env.WORKSPACE}/kubeconfig-${jobID}"
                        oc login \${OC_ADDRESS} --username=\${OC_USERNAME} --password=\${OC_PASSWORD} --insecure-skip-tls-verify=true
                        oc delete --cascade=foreground -f test/env/${testEnv.beegfsHost}/csi-beegfs-cr.yaml || true
                        operator-sdk cleanup beegfs-csi-driver-operator || true
                        while [ "\$(oc get pod | grep beegfs-csi-driver-operator)" ]; do 
                            echo "waiting for bundle cleanup"
                            sleep 5
                        done
                        operator-sdk scorecard ./operator/bundle -w 180s > ${resultsDir}/scorecard.txt 2>&1 || (echo "SCORECARD FAILURE!" && exit 1)
                        operator-sdk run bundle ${uniqueBundleImageTag}
                        sed 's/tag: replaced-by-jenkins/tag: ${uniqueImageTag.split(':')[1]}/g' test/env/${testEnv.beegfsHost}/csi-beegfs-cr.yaml | kubectl apply -f -
                        export KUBE_SSH_USER=\${SSH_OPENSHIFT_USR}
                        export KUBE_SSH_KEY_PATH=\${SSH_OPENSHIFT}
                        ${testCommand} || (echo "INTEGRATION TEST FAILURE!" && exit 1)
                        ${testCommandDisruptive} || (echo "DISRUPTIVE INTEGRATION TEST FAILURE!" && exit 1)
                    """
                } finally {
                    sh """
                        export KUBECONFIG="${env.WORKSPACE}/kubeconfig-${jobID}"
                        oc get ns --no-headers | awk '{print \$1}' | grep -e provisioning- -e stress- -e beegfs- -e multivolume- -e ephemeral- -e volumemode- |
                            grep -v beegfs-csi | xargs kubectl delete ns --cascade=foreground || true
                        oc delete -f test/env/${testEnv.beegfsHost}/csi-beegfs-cr.yaml || true
                        operator-sdk cleanup beegfs-csi-driver-operator || true
                    """
                    // Use junit here (on a per-environment basis) instead of once in post so Jenkins visualizer makes
                    // it clear which environment failed.
                    junit "${resultsDir}/*.xml"
                }
            }
        } else {
            // Credentials variables are always local to the withCredentials block, so multiple
            // instances of the KUBECONFIG variable can exist without issue when running in parallel.
            withCredentials([file(credentialsId: "kubeconfig-${testEnv.k8sCluster}", variable: "KUBECONFIG")]) {
                def overlay = "deploy/k8s/overlays/${jobID}"
                sh """
                    cp -r deploy/k8s/overlays/default ${overlay}
                    (cd ${overlay} && \\
                    kustomize edit set image netapp/beegfs-csi-driver=${uniqueImageTag} && \\
                    sed -i 's?/versions/latest?/versions/v${testEnv.k8sVersion}?g' kustomization.yaml)
                """
                try {
                    // The two kubectl get ... lines are used to clean up any beegfs CSI driver currently
                    // running on the cluster. We can't simply delete using -k deploy/k8s/overlay-xxx/ because a previous
                    // user might have deployed the driver using a different deployment scheme.
                    //
                    // It's a bit awkward to include Scorecard testing here, because Scorecard currently only evaluates
                    // our bundle (doesn't really do integration tests) and currently isn't expected to get different
                    // results on different clusters. However:
                    //   1) We need kubeconfig access to a cluster to run Scorecard.
                    //   2) We may eventually write custom tests with cluster-dependent results.
                    sh """                   
                        kubectl get sts -A | grep csi-beegfs | awk '{print \$2 " -n " \$1}' | xargs kubectl delete --cascade=foreground sts || true
                        kubectl get ds -A | grep csi-beegfs | awk '{print \$2 " -n " \$1}' | xargs kubectl delete --cascade=foreground ds || true
                        operator-sdk scorecard ./operator/bundle -w 180s > ${resultsDir}/scorecard.txt 2>&1 || (echo "SCORECARD FAILURE!" && exit 1)
                        cp test/env/${testEnv.beegfsHost}/csi-beegfs-config.yaml ${overlay}/csi-beegfs-config.yaml
                        cp test/env/${testEnv.beegfsHost}/csi-beegfs-connauth.yaml ${overlay}/csi-beegfs-connauth.yaml
                        kubectl apply -k ${overlay}
                        export KUBE_SSH_USER=${testEnv.sshUser}
                        ${testCommand} || (echo "INTEGRATION TEST FAILURE!" && exit 1)
                        ${testCommandDisruptive} || (echo "DISRUPTIVE INTEGRATION TEST FAILURE!" && exit 1)
                    """
                } finally {
                    sh """
                        kubectl get ns --no-headers | awk '{print \$1}' | grep -e provisioning- -e stress- -e beegfs- -e multivolume- -e ephemeral- -e volumemode- |
                            grep -v beegfs-csi | xargs kubectl delete ns --cascade=foreground || true
                        kubectl delete --cascade=foreground -k ${overlay} || true
                    """
                    // Use junit here (on a per-environment basis) instead of once in post so Jenkins visualizer makes
                    // it clear which environment failed.
                    junit "${resultsDir}/*.xml"
                }
            }
        }
    }
}

// TestEnvironment is a JavaBean type class used only to store data about a particular test environment. It mainly
// exists to allow storing strings AND booleans describing a test environment in one structure.
class TestEnvironment {
    String k8sCluster
    String beegfsHost
    String k8sVersion
    String staticVolDirName
    String sshUser
    // useOperator could somewhat equivalently be called inOpenShift. When this is true, we assume testing occurs in an
    // OpenShift cluster (which carries certain extra burdens) AND the driver should be deployed using the operator and
    // OLM. If we ever increase our test coverage so that we use OLM in a Kubesprayed cluster or deploy to OpenShift
    // without OLM, we may need to decouple this field into useOperator and inOpenShift.
    boolean useOperator

    TestEnvironment(String k8sCluster, String beegfsHost, String k8sVersion, String staticVolDirName, String sshUser, boolean useOperator) {
        this.k8sCluster = k8sCluster
        this.beegfsHost = beegfsHost
        this.k8sVersion = k8sVersion
        this.staticVolDirName = staticVolDirName
        this.sshUser = sshUser
        this.useOperator = useOperator
    }
}
