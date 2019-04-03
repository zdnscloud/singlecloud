pipeline {
  agent any
  stages {
    stage('sync submodule') {
      steps {
        sh 'git submodule update --init'
      }
    }
    stage('make ui') {
      steps {
        sh 'cd ui && make build-image'
      }
    }
    stage('make zcloud') {
      steps {
        sh 'make docker'
      }
    }
  }
}