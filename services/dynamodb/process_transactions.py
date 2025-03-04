import json
import boto3
from decimal import Decimal
from datetime import datetime
import math
import boto3.session

def load_json_file(file_path):
    with open(file_path, 'r') as file:
        return json.load(file)

def format_for_dynamodb(transaction):
    formatted_item = {}
    for key, value in transaction.items():
        if isinstance(value, (int, float)):
            formatted_item[key] = Decimal(str(value))
        else:
            formatted_item[key] = value
            
    return {
        'PutRequest': {
            'Item': formatted_item
        }
    }

def chunk_list(lst, chunk_size):
    """Split list into chunks of specified size"""
    return [lst[i:i + chunk_size] for i in range(0, len(lst), chunk_size)]

def batch_write_to_dynamodb(transactions, table_name):
    session = boto3.Session(
        profile_name='CS620_C1_Capstone_Rex',
        region_name='us-east-1'
    )
    dynamodb = session.resource('dynamodb')
    table = dynamodb.Table(table_name)
    
    formatted_transactions = [format_for_dynamodb(tx) for tx in transactions]
    
    chunked_transactions = chunk_list(formatted_transactions, 25)
    
    for i, chunk in enumerate(chunked_transactions):
        try:
            response = dynamodb.batch_write_item(
                RequestItems={
                    table_name: chunk
                }
            )
            print(f"Successfully processed batch {i+1} of {len(chunked_transactions)}")
            
            unprocessed = response.get('UnprocessedItems', {})
            while unprocessed:
                response = dynamodb.batch_write_item(RequestItems=unprocessed)
                unprocessed = response.get('UnprocessedItems', {})
                
        except Exception as e:
            print(f"Error processing batch {i+1}: {str(e)}")

def main():
    TABLE_NAME = 'Transaction_Information'
    JSON_FILE_PATH = 'bank_transactions_data.json'
    
    try:
        transactions = load_json_file(JSON_FILE_PATH)
        
        total_batches = math.ceil(len(transactions) / 25)
        print(f"Total transactions: {len(transactions)}")
        print(f"Total batches to process: {total_batches}")
        
        batch_write_to_dynamodb(transactions, TABLE_NAME)
        
        print("Data processing completed successfully!")
        
    except Exception as e:
        print(f"Error: {str(e)}")

if __name__ == "__main__":
    main() 