#!/usr/bin/env python3

import os
import sys
import argparse
import pickle
import subprocess
import json
from googleapiclient.discovery import build
from google_auth_oauthlib.flow import InstalledAppFlow
from google.auth.transport.requests import Request
from googleapiclient.http import MediaFileUpload
from googleapiclient.errors import HttpError

SCOPES = ['https://www.googleapis.com/auth/youtube.upload', 'https://www.googleapis.com/auth/youtube.readonly']
API_SERVICE_NAME = 'youtube'
API_VERSION = 'v3'
CLIENT_SECRETS_FILE = 'client_secret.json'
TOKEN_FILE = 'token.pickle'

def authenticate_youtube():
    """Authenticate with YouTube API and return service object."""
    creds = None
    
    if os.path.exists(TOKEN_FILE):
        with open(TOKEN_FILE, 'rb') as token:
            creds = pickle.load(token)
    
    if not creds or not creds.valid:
        if creds and creds.expired and creds.refresh_token:
            creds.refresh(Request())
        else:
            if not os.path.exists(CLIENT_SECRETS_FILE):
                print(f"Error: {CLIENT_SECRETS_FILE} not found. Please download from Google Cloud Console.")
                sys.exit(1)
            
            flow = InstalledAppFlow.from_client_secrets_file(
                CLIENT_SECRETS_FILE, SCOPES)
            creds = flow.run_local_server(port=0)
        
        with open(TOKEN_FILE, 'wb') as token:
            pickle.dump(creds, token)
    
    return build(API_SERVICE_NAME, API_VERSION, credentials=creds)

def get_video_dimensions(video_file):
    """Get video dimensions using ffprobe."""
    try:
        cmd = [
            'ffprobe', '-v', 'quiet', '-print_format', 'json',
            '-show_streams', '-select_streams', 'v:0', video_file
        ]
        result = subprocess.run(cmd, capture_output=True, text=True, check=True)
        data = json.loads(result.stdout)
        
        if 'streams' in data and len(data['streams']) > 0:
            stream = data['streams'][0]
            width = int(stream.get('width', 0))
            height = int(stream.get('height', 0))
            return width, height
    except (subprocess.CalledProcessError, json.JSONDecodeError, KeyError, ValueError) as e:
        print(f"Warning: Could not detect video dimensions: {e}")
    
    return None, None

def is_vertical_short(width, height):
    """Check if video is a vertical short (1080x1920)."""
    return width == 1080 and height == 1920

def verify_channel_access(youtube, channel_id):
    """Verify that the authenticated user has access to the specified channel."""
    try:
        request = youtube.channels().list(
            part="id,snippet",
            id=channel_id
        )
        response = request.execute()
        
        if not response.get('items'):
            print(f"Error: Channel ID '{channel_id}' not found or not accessible.")
            return False
        
        channel = response['items'][0]
        print(f"Uploading to channel: {channel['snippet']['title']}")
        return True
    
    except HttpError as e:
        print(f"Error verifying channel access: {e}")
        return False

def upload_video(youtube, video_file, title, description, channel_id=None, tags=None, category_id="22", privacy_status="private", thumbnail_path=None, not_made_for_kids=True, related_video_id=None):
    """Upload a video to YouTube."""
    if not os.path.exists(video_file):
        print(f"Error: Video file '{video_file}' not found.")
        return None
    
    if channel_id and not verify_channel_access(youtube, channel_id):
        return None
    
    tags = tags or []
    
    # Check if this is a vertical short and use related video if specified
    width, height = get_video_dimensions(video_file)
    if is_vertical_short(width, height) and related_video_id:
        print(f"Detected 1080x1920 short, adding related video: {related_video_id}")
        # Add related video to description
        if description:
            description += f"\n\nRelated video: https://www.youtube.com/watch?v={related_video_id}"
        else:
            description = f"Related video: https://www.youtube.com/watch?v={related_video_id}"
    
    body = {
        'snippet': {
            'title': title,
            'description': description,
            'tags': tags,
            'categoryId': category_id
        },
        'status': {
            'privacyStatus': privacy_status,
            'selfDeclaredMadeForKids': False,
        }
    }
    
    if channel_id:
        body['snippet']['channelId'] = channel_id
    
    media = MediaFileUpload(video_file, chunksize=-1, resumable=True)
    
    try:
        insert_request = youtube.videos().insert(
            part=','.join(body.keys()),
            body=body,
            media_body=media
        )
        
        print(f"Uploading video: {title}")
        response = None
        error = None
        retry = 0
        
        while response is None:
            try:
                print("Uploading file...")
                status, response = insert_request.next_chunk()
                if status:
                    print(f"Uploaded {int(status.progress() * 100)}%")
            except HttpError as e:
                if e.resp.status in [500, 502, 503, 504]:
                    error = f"A retriable HTTP error {e.resp.status} occurred:\n{e.content}"
                    retry += 1
                    if retry > 5:
                        print("Too many retry attempts.")
                        return None
                else:
                    raise
            except Exception as e:
                print(f"An error occurred: {e}")
                return None
        
        if response is not None:
            if 'id' in response:
                video_id = response['id']
                print(f"Video uploaded successfully!")
                print(f"Video ID: {video_id}")
                print(f"Video URL: https://www.youtube.com/watch?v={video_id}")
                
                # Upload thumbnail if provided
                if thumbnail_path and os.path.exists(thumbnail_path):
                    upload_thumbnail(youtube, video_id, thumbnail_path)
                elif thumbnail_path:
                    print(f"Warning: Thumbnail file '{thumbnail_path}' not found.")
                
                return video_id
            else:
                print(f"Upload failed with unexpected response: {response}")
                return None
    
    except HttpError as e:
        print(f"An HTTP error {e.resp.status} occurred: {e.content}")
        return None

def upload_thumbnail(youtube, video_id, thumbnail_path):
    """Upload a thumbnail for the specified video."""
    try:
        print(f"Uploading thumbnail: {thumbnail_path}")
        media = MediaFileUpload(thumbnail_path)
        
        request = youtube.thumbnails().set(
            videoId=video_id,
            media_body=media
        )
        
        response = request.execute()
        print("Thumbnail uploaded successfully!")
        return response
        
    except HttpError as e:
        print(f"An HTTP error occurred while uploading thumbnail: {e}")
        return None
    except Exception as e:
        print(f"An error occurred while uploading thumbnail: {e}")
        return None

def main():
    parser = argparse.ArgumentParser(description='Upload video to YouTube')
    parser.add_argument('video_file', help='Path to video file')
    parser.add_argument('--title', required=True, help='Video title')
    parser.add_argument('--description', default='', help='Path to file containing video description')
    parser.add_argument('--tags', help='Comma-separated list of tags')
    parser.add_argument('--category', default='26', help='YouTube category ID (default: 22 for People & Blogs)')
    parser.add_argument('--privacy', choices=['private', 'public', 'unlisted'], default='private', help='Privacy status')
    parser.add_argument('--channel-id', help='YouTube channel ID to upload to (required if you have multiple channels)')
    parser.add_argument('--thumbnail', help='Path to PNG thumbnail image file')
    parser.add_argument('--not-made-for-kids', action='store_true', help='Mark video as not made for kids')
    parser.add_argument('--related-video-id', help='Related video ID to add for 1080x1920 shorts (e.g., h8EfnJmcwG4)')
    
    args = parser.parse_args()
    
    tags = args.tags.split(',') if args.tags else []
    tags = [tag.strip() for tag in tags]
    
    # Load description from file if provided
    description = ''
    if args.description:
        try:
            with open(args.description, 'r', encoding='utf-8') as f:
                description = f.read().strip()
        except FileNotFoundError:
            print(f"Error: Description file '{args.description}' not found.")
            sys.exit(1)
        except Exception as e:
            print(f"Error reading description file '{args.description}': {e}")
            sys.exit(1)
    
    try:
        youtube = authenticate_youtube()
        video_id = upload_video(
            youtube=youtube,
            video_file=args.video_file,
            title=args.title,
            description=description,
            channel_id=getattr(args, 'channel_id'),
            tags=tags,
            category_id=args.category,
            privacy_status=args.privacy,
            thumbnail_path=args.thumbnail,
            not_made_for_kids=args.not_made_for_kids,
            related_video_id=getattr(args, 'related_video_id')
        )
        
        if video_id:
            print(f"Upload completed successfully. Video ID: {video_id}")
        else:
            print("Upload failed.")
            sys.exit(1)
            
    except Exception as e:
        print(f"Error: {e}")
        sys.exit(1)

if __name__ == '__main__':
    main()
