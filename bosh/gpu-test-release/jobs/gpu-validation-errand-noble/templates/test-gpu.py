#!/usr/bin/env python3
"""
GPU Validation Test Script

Simple validation that GPU compute works with PyTorch and TensorFlow.
Tests:
  1. GPU detection
  2. FP32 matrix multiplication (baseline compute)
  3. FP16 matrix multiplication (Tensor Core validation)
"""

import argparse
import json
import sys
import time
from datetime import datetime

# Try to import both frameworks
PYTORCH_AVAILABLE = False
TENSORFLOW_AVAILABLE = False

try:
    import torch
    PYTORCH_AVAILABLE = True
except ImportError:
    pass

try:
    import tensorflow as tf
    TENSORFLOW_AVAILABLE = True
except ImportError:
    pass


###############################################################################
# PyTorch Implementation
###############################################################################

def pytorch_get_gpu_info():
    """Get GPU information using PyTorch"""
    if not torch.cuda.is_available():
        return None

    return {
        "cuda_available": True,
        "device_count": torch.cuda.device_count(),
        "device_name": torch.cuda.get_device_name(0),
        "cuda_version": torch.version.cuda,
        "pytorch_version": torch.__version__,
        "memory_total_mb": torch.cuda.get_device_properties(0).total_memory / (1024**2),
    }


def pytorch_test_matmul(size, iterations, dtype):
    """Test matrix multiplication performance"""
    device = torch.device("cuda")

    if dtype == "fp32":
        a = torch.randn(size, size, device=device, dtype=torch.float32)
        b = torch.randn(size, size, device=device, dtype=torch.float32)
    else:  # fp16
        a = torch.randn(size, size, device=device, dtype=torch.float16)
        b = torch.randn(size, size, device=device, dtype=torch.float16)

    # Warmup
    for _ in range(3):
        _ = torch.matmul(a, b)
    torch.cuda.synchronize()

    # Timed runs
    times = []
    for _ in range(iterations):
        start = time.perf_counter()
        _ = torch.matmul(a, b)
        torch.cuda.synchronize()
        times.append(time.perf_counter() - start)

    avg_time = sum(times) / len(times)
    flops = 2 * (size ** 3)
    tflops = (flops / avg_time) / 1e12

    return {
        "dtype": dtype,
        "matrix_size": size,
        "iterations": iterations,
        "avg_time_seconds": avg_time,
        "tflops": tflops,
    }


###############################################################################
# TensorFlow Implementation
###############################################################################

def tensorflow_get_gpu_info():
    """Get GPU information using TensorFlow"""
    gpus = tf.config.list_physical_devices('GPU')
    if not gpus:
        return None

    return {
        "cuda_available": True,
        "device_count": len(gpus),
        "device_name": gpus[0].name if gpus else "unknown",
        "tensorflow_version": tf.__version__,
    }


def tensorflow_test_matmul(size, iterations, dtype):
    """Test matrix multiplication performance"""
    with tf.device('/GPU:0'):
        if dtype == "fp32":
            a = tf.random.normal([size, size], dtype=tf.float32)
            b = tf.random.normal([size, size], dtype=tf.float32)
        else:  # fp16
            a = tf.random.normal([size, size], dtype=tf.float16)
            b = tf.random.normal([size, size], dtype=tf.float16)

        # Warmup
        for _ in range(3):
            _ = tf.matmul(a, b)

        # Timed runs
        times = []
        for _ in range(iterations):
            start = time.perf_counter()
            c = tf.matmul(a, b)
            _ = c.numpy()  # Force execution
            times.append(time.perf_counter() - start)

    avg_time = sum(times) / len(times)
    flops = 2 * (size ** 3)
    tflops = (flops / avg_time) / 1e12

    return {
        "dtype": dtype,
        "matrix_size": size,
        "iterations": iterations,
        "avg_time_seconds": avg_time,
        "tflops": tflops,
    }


###############################################################################
# Main Test Runner
###############################################################################

def run_tests(framework, matrix_size, iterations, output_file):
    """Run GPU tests for the specified framework"""

    results = {
        "timestamp": datetime.utcnow().isoformat() + "Z",
        "framework": framework,
        "tests": {},
        "passed": False,
    }

    # Select framework functions
    if framework == "pytorch":
        if not PYTORCH_AVAILABLE:
            results["error"] = "PyTorch not available"
            with open(output_file, 'w') as f:
                json.dump(results, f, indent=2)
            return False

        get_gpu_info = pytorch_get_gpu_info
        test_matmul = pytorch_test_matmul

    elif framework == "tensorflow":
        if not TENSORFLOW_AVAILABLE:
            results["error"] = "TensorFlow not available"
            with open(output_file, 'w') as f:
                json.dump(results, f, indent=2)
            return False

        get_gpu_info = tensorflow_get_gpu_info
        test_matmul = tensorflow_test_matmul

    else:
        results["error"] = f"Unknown framework: {framework}"
        with open(output_file, 'w') as f:
            json.dump(results, f, indent=2)
        return False

    # Test 1: GPU Detection
    print("Test 1: GPU Detection")
    gpu_info = get_gpu_info()
    if gpu_info is None:
        print("FAILED: No GPU detected")
        results["error"] = "No GPU detected"
        with open(output_file, 'w') as f:
            json.dump(results, f, indent=2)
        return False

    print(f"  Device: {gpu_info.get('device_name', 'unknown')}")
    print(f"  CUDA available: {gpu_info.get('cuda_available', False)}")
    results["gpu_info"] = gpu_info
    results["tests"]["gpu_detection"] = "PASSED"

    # Test 2: FP32 Matrix Multiplication
    print(f"\nTest 2: FP32 Matrix Multiplication ({matrix_size}x{matrix_size})")
    try:
        fp32_results = test_matmul(matrix_size, iterations, "fp32")
        print(f"  Average time: {fp32_results['avg_time_seconds']*1000:.2f} ms")
        print(f"  Performance: {fp32_results['tflops']:.2f} TFLOPS")
        results["tests"]["matmul_fp32"] = fp32_results
        results["tests"]["matmul_fp32"]["status"] = "PASSED"
    except Exception as e:
        print(f"FAILED: {e}")
        results["tests"]["matmul_fp32"] = {"status": "FAILED", "error": str(e)}

    # Test 3: FP16 Matrix Multiplication (Tensor Cores)
    print(f"\nTest 3: FP16 Matrix Multiplication - Tensor Cores ({matrix_size}x{matrix_size})")
    try:
        fp16_results = test_matmul(matrix_size, iterations, "fp16")
        print(f"  Average time: {fp16_results['avg_time_seconds']*1000:.2f} ms")
        print(f"  Performance: {fp16_results['tflops']:.2f} TFLOPS")

        # Calculate speedup
        if "matmul_fp32" in results["tests"] and results["tests"]["matmul_fp32"].get("tflops"):
            speedup = fp16_results["tflops"] / results["tests"]["matmul_fp32"]["tflops"]
            fp16_results["speedup_vs_fp32"] = speedup
            print(f"  Speedup vs FP32: {speedup:.1f}x")

        results["tests"]["matmul_fp16"] = fp16_results
        results["tests"]["matmul_fp16"]["status"] = "PASSED"
    except Exception as e:
        print(f"FAILED: {e}")
        results["tests"]["matmul_fp16"] = {"status": "FAILED", "error": str(e)}

    # Overall result
    all_passed = all(
        t.get("status") == "PASSED"
        for t in results["tests"].values()
        if isinstance(t, dict)
    )
    results["passed"] = all_passed

    # Write results
    with open(output_file, 'w') as f:
        json.dump(results, f, indent=2)

    print(f"\n{'='*50}")
    if all_passed:
        print("ALL TESTS PASSED")
        return True
    else:
        print("SOME TESTS FAILED")
        return False


def main():
    parser = argparse.ArgumentParser(description='GPU Validation Tests')
    parser.add_argument('--framework', type=str, default='pytorch',
                        choices=['pytorch', 'tensorflow', 'both'],
                        help='ML framework to test')
    parser.add_argument('--matrix-size', type=int, default=4096,
                        help='Matrix size for tests')
    parser.add_argument('--iterations', type=int, default=10,
                        help='Number of iterations')
    parser.add_argument('--output', type=str, default='results.json',
                        help='Output JSON file')
    args = parser.parse_args()

    if args.framework == 'both':
        # Run both frameworks
        frameworks = []
        if PYTORCH_AVAILABLE:
            frameworks.append('pytorch')
        if TENSORFLOW_AVAILABLE:
            frameworks.append('tensorflow')

        if not frameworks:
            print("ERROR: Neither PyTorch nor TensorFlow is available")
            sys.exit(1)

        all_passed = True
        for fw in frameworks:
            print(f"\n{'='*60}")
            print(f"Running {fw.upper()} tests")
            print(f"{'='*60}\n")

            output_file = args.output.replace('.json', f'-{fw}.json')
            passed = run_tests(fw, args.matrix_size, args.iterations, output_file)
            all_passed = all_passed and passed

        sys.exit(0 if all_passed else 1)

    else:
        # Run single framework
        passed = run_tests(args.framework, args.matrix_size, args.iterations, args.output)
        sys.exit(0 if passed else 1)


if __name__ == "__main__":
    main()
