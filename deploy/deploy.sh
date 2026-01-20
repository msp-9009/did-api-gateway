#!/bin/bash
# Deployment script for Phase 1 & Phase 2 - Privacy-Preserving DID Gateway

set -e

echo "=================================="
echo "DID Gateway Deployment Script"
echo "Phase 1: Security Hardening"
echo "Phase 2: DID Resolution Enhancement"
echo "=================================="
echo ""

# Configuration
NAMESPACE="${K8S_NAMESPACE:-default}"
ENVIRONMENT="${ENVIRONMENT:-dev}"
DOCKER_REGISTRY="${DOCKER_REGISTRY:-localhost:5000}"
VERSION="${VERSION:-latest}"

# Colors for output
RED='\033[0:31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Helper functions
function info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

function warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

function error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

function check_prerequisites() {
    info "Checking prerequisites..."
    
    # Check docker
    if ! command -v docker &> /dev/null; then
        error "Docker not found. Please install Docker."
    fi
    
    # Check kubectl
    if ! command -v kubectl &> /dev/null; then
        error "kubectl not found. Please install kubectl."
    fi
    
    # Check cluster connection
    if ! kubectl cluster-info &> /dev/null; then
        error "Cannot connect to Kubernetes cluster."
    fi
    
    info "Prerequisites check passed ✓"
}

function generate_secrets() {
    info "Generating secrets if not exists..."
    
    if kubectl get secret gateway-secrets -n "$NAMESPACE" &> /dev/null; then
        warn "Secrets already exist. Skipping generation."
    else
        info "Running secret generation script..."
        cd deploy/k8s
        ./generate-secrets.sh
        cd ../..
        
        info "Applying secrets to cluster..."
        kubectl apply -f deploy/k8s/generated-secrets.yaml -n "$NAMESPACE"
    fi
    
    info "Secrets configured ✓"
}

function build_images() {
    info "Building Docker images..."
    
    # Build Gateway
    info "Building Gateway image..."
    docker build -f deploy/docker/Dockerfile.gateway \
        -t "${DOCKER_REGISTRY}/did-gateway:${VERSION}" .
    
    # Build Issuer
    info "Building Issuer image..."
    docker build -f deploy/docker/Dockerfile.issuer \
        -t "${DOCKER_REGISTRY}/did-issuer:${VERSION}" .
    
    # Build Upstream
    info "Building Upstream image..."
    docker build -f deploy/docker/Dockerfile.upstream \
        -t "${DOCKER_REGISTRY}/did-upstream:${VERSION}" .
    
    info "Images built ✓"
}

function push_images() {
    info "Pushing images to registry..."
    
    docker push "${DOCKER_REGISTRY}/did-gateway:${VERSION}"
    docker push "${DOCKER_REGISTRY}/did-issuer:${VERSION}"
    docker push "${DOCKER_REGISTRY}/did-upstream:${VERSION}"
    
    info "Images pushed ✓"
}

function deploy_infrastructure() {
    info "Deploying infrastructure components..."
    
    # Deploy PostgreSQL
    info "Deploying PostgreSQL..."
    kubectl apply -f deploy/k8s/postgres.yaml -n "$NAMESPACE"
    
    # Deploy Redis
    info "Deploying Redis..."
    kubectl apply -f deploy/k8s/redis.yaml -n "$NAMESPACE"
    
    # Wait for databases to be ready
    info "Waiting for databases to be ready..."
    kubectl wait --for=condition=ready pod -l app=postgres -n "$NAMESPACE" --timeout=300s
    kubectl wait --for=condition=ready pod -l app=redis -n "$NAMESPACE" --timeout=300s
    
    info "Infrastructure deployed ✓"
}

function install_cert_manager() {
    info "Checking cert-manager installation..."
    
    if kubectl get namespace cert-manager &> /dev/null; then
        warn "cert-manager already installed. Skipping..."
    else
        info "Installing cert-manager..."
        kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml
        
        info "Waiting for cert-manager to be ready..."
        kubectl wait --for=condition=Available deployment -l app.kubernetes.io/instance=cert-manager \
            -n cert-manager --timeout=300s
    fi
    
    # Apply certificate configuration
    info "Applying certificate configuration..."
    kubectl apply -f deploy/k8s/certificates.yaml -n "$NAMESPACE"
    
    info "cert-manager configured ✓"
}

function deploy_services() {
    info "Deploying application services..."
    
    # Update image tags in manifests
    sed -i "s|image: did-gateway:latest|image: ${DOCKER_REGISTRY}/did-gateway:${VERSION}|g" deploy/k8s/gateway.yaml
    sed -i "s|image: did-issuer:latest|image: ${DOCKER_REGISTRY}/did-issuer:${VERSION}|g" deploy/k8s/issuer.yaml
    sed -i "s|image: did-upstream:latest|image: ${DOCKER_REGISTRY}/did-upstream:${VERSION}|g" deploy/k8s/upstream.yaml
    
    # Deploy Upstream
    info "Deploying Upstream service..."
    kubectl apply -f deploy/k8s/upstream.yaml -n "$NAMESPACE"
    
    # Deploy Issuer
    info "Deploying Issuer service..."
    kubectl apply -f deploy/k8s/issuer.yaml -n "$NAMESPACE"
    
    # Deploy Gateway
    info "Deploying Gateway service..."
    kubectl apply -f deploy/k8s/gateway.yaml -n "$NAMESPACE"
    
    # Deploy Ingress
    info "Deploying Ingress..."
    kubectl apply -f deploy/k8s/ingress.yaml -n "$NAMESPACE"
    
    info "Services deployed ✓"
}

function wait_for_deployment() {
    info "Waiting for deployments to be ready..."
    
    kubectl wait --for=condition=available deployment/did-upstream -n "$NAMESPACE" --timeout=300s
    kubectl wait --for=condition=available deployment/did-issuer -n "$NAMESPACE" --timeout=300s
    kubectl wait --for=condition=available deployment/did-gateway -n "$NAMESPACE" --timeout=300s
    
    info "All deployments ready ✓"
}

function run_smoke_tests() {
    info "Running smoke tests..."
    
    # Get gateway pod
    GATEWAY_POD=$(kubectl get pod -l app=did-gateway -n "$NAMESPACE" -o jsonpath='{.items[0].metadata.name}')
    
    # Test health endpoint
    info "Testing health endpoint..."
    kubectl exec -n "$NAMESPACE" "$GATEWAY_POD" -- wget -q -O- http://localhost:8080/healthz || error "Health check failed"
    
    # Test readiness endpoint
    info "Testing readiness endpoint..."
    kubectl exec -n "$NAMESPACE" "$GATEWAY_POD" -- wget -q -O- http://localhost:8080/readyz || error "Readiness check failed"
    
    info "Smoke tests passed ✓"
}

function print_status() {
    info "Deployment Status:"
    echo ""
    kubectl get pods -n "$NAMESPACE" -l 'app in (did-gateway,did-issuer,did-upstream,postgres,redis)'
    echo ""
    kubectl get svc -n "$NAMESPACE" -l 'app in (did-gateway,did-issuer,did-upstream)'
    echo ""
    kubectl get ingress -n "$NAMESPACE"
    echo ""
    
    info "Deployment complete! 🎉"
    echo ""
    echo "Gateway endpoint: https://$(kubectl get ingress gateway-ingress -n "$NAMESPACE" -o jsonpath='{.spec.rules[0].host}')"
    echo ""
}

# Main deployment flow
function main() {
    info "Starting deployment to namespace: $NAMESPACE"
    info "Environment: $ENVIRONMENT"
    echo ""
    
    check_prerequisites
    generate_secrets
    build_images
    
    if [ "$ENVIRONMENT" != "dev" ]; then
        push_images
    fi
    
    deploy_infrastructure
    
    if [ "$ENVIRONMENT" = "production" ] || [ "$ENVIRONMENT" = "staging" ]; then
        install_cert_manager
    fi
    
    deploy_services
    wait_for_deployment
    run_smoke_tests
    print_status
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --namespace|-n)
            NAMESPACE="$2"
            shift 2
            ;;
        --environment|-e)
            ENVIRONMENT="$2"
            shift 2
            ;;
        --version|-v)
            VERSION="$2"
            shift 2
            ;;
        --skip-build)
            SKIP_BUILD=true
            shift
            ;;
        --help|-h)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  -n, --namespace NAMESPACE    Kubernetes namespace (default: default)"
            echo "  -e, --environment ENV        Environment: dev/staging/production (default: dev)"
            echo "  -v, --version VERSION        Image version (default: latest)"
            echo "  --skip-build                 Skip Docker image building"
            echo "  -h, --help                   Show this help message"
            exit 0
            ;;
        *)
            error "Unknown option: $1. Use --help for usage information."
            ;;
    esac
done

# Run main deployment
main
