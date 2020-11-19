# Jenkinsfile Runner Operator
Operator for [jenkinsfile-runner](https://github.com/jenkinsci/jenkinsfile-runner) to manage and run runner instances easily.

## Overview
The Operator exposes the jenkinsfilerunner.io/v1alpha1 `RunnerImage`, `Runner` and `Run` APIs for Kubernetes to create and run serverless Jenkins pipelines.

### Runs
An instance of a Jenkins Pipeline execution which would be created against a `Runner`. 
A `Runner` has to already be present and be in a running state for the `Run` to execute.

### Runner
A running `Pod` which is running the `jenkins/jenkinsfile-runner` image. `Runs` would be executed against this `Pod`.