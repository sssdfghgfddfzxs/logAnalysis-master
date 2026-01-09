#!/usr/bin/env python3
"""
Script to generate Python gRPC bindings from protobuf files
"""

import subprocess
import sys
import os

def generate_proto():
    """Generate Python gRPC bindings"""
    proto_dir = "proto"
    output_dir = "src/proto"
    
    # Create output directory if it doesn't exist
    os.makedirs(output_dir, exist_ok=True)
    
    # Create __init__.py file
    init_file = os.path.join(output_dir, "__init__.py")
    with open(init_file, "w") as f:
        f.write("# Generated protobuf files\n")
    
    # Generate Python bindings
    proto_file = os.path.join(proto_dir, "ai_analysis.proto")
    
    if not os.path.exists(proto_file):
        print(f"Error: {proto_file} not found")
        return False
    
    cmd = [
        sys.executable, "-m", "grpc_tools.protoc",
        f"--proto_path={proto_dir}",
        f"--python_out={output_dir}",
        f"--grpc_python_out={output_dir}",
        proto_file
    ]
    
    try:
        result = subprocess.run(cmd, check=True, capture_output=True, text=True)
        print("Successfully generated Python gRPC bindings")
        print(f"Generated files in {output_dir}/")
        return True
    except subprocess.CalledProcessError as e:
        print(f"Error generating protobuf files: {e}")
        print(f"stdout: {e.stdout}")
        print(f"stderr: {e.stderr}")
        return False

if __name__ == "__main__":
    success = generate_proto()
    sys.exit(0 if success else 1)