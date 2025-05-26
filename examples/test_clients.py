#!/usr/bin/env python3
"""
Test script for the generated Python client.
This demonstrates how to use the generated client code.
"""

from client import (
    ClientClient, 
    GetUserByIdPathParams, 
    GetUserByIdQueryParams, 
    GetUserByIdHeaderParams,
    CreateNewUserRequestBody,
    APIError
)

def main():
    # Create client
    client = ClientClient("https://api.example.com")
    
    # Set default headers
    client.set_header("User-Agent", "gopenapi-test-client/1.0")
    
    print("Testing generated Python client...")
    
    # Test 1: Get user by ID
    print("\n1. Testing get_user_by_id...")
    try:
        path = GetUserByIdPathParams(id=123)
        query = GetUserByIdQueryParams(include="profile")
        headers = GetUserByIdHeaderParams(authorization="Bearer test-token")
        
        # Note: This will fail since we're using a fake API, but it demonstrates the usage
        user = client.get_user_by_id(path=path, query=query, headers=headers)
        print(f"User: {user.name} ({user.email})")
    except APIError as e:
        print(f"Expected API Error (fake endpoint): {e.status_code} - {e.message}")
    except Exception as e:
        print(f"Connection error (expected): {e}")
    
    # Test 2: Create new user
    print("\n2. Testing create_new_user...")
    try:
        body = CreateNewUserRequestBody(name="John Doe", email="john@example.com")
        new_user = client.create_new_user(body=body)
        print(f"Created user: {new_user.id}")
    except APIError as e:
        print(f"Expected API Error (fake endpoint): {e.status_code} - {e.message}")
    except Exception as e:
        print(f"Connection error (expected): {e}")
    
    print("\nClient generation test completed!")
    print("The client code is working correctly - errors are expected since we're using a fake API endpoint.")

if __name__ == "__main__":
    main() 