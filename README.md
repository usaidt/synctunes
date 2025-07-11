# SyncTunes 🎵

**Host synchronized music listening parties with your friends, anywhere in the world.**

SyncTunes is a self-hosted collaborative music streaming application that lets you create virtual listening rooms where everyone hears the same song at exactly the same time. Perfect for remote hangouts, music discovery sessions, or just sharing your favorite tracks with friends.

## ✨ What Makes SyncTunes Special

**Real-time Synchronization** - Everyone in the room hears the exact same moment of each song, no matter where they are in the world.

**Your Music, Your Rules** - Host your own music collection without relying on streaming service subscriptions or playlists.

**Instant Sharing** - Create a room and share the link. No accounts, no downloads, no complicated setup for your friends.

**Lightweight & Fast** - Built with Go for minimal server resources and lightning-fast response times.

**Modern Interface** - Clean, responsive design that works great on phones, tablets, and desktops.

## 🚀 Getting Started

### Option 1: Docker (Easiest)

If you have Docker installed, you can be up and running in under 2 minutes:

```bash
git clone https://github.com/usaidt/synctunes.git
cd synctunes
docker-compose up -d
```

Add your music files to the `music/` folder, then visit `http://localhost:8080` to start hosting!

### Option 2: Direct Installation

If you prefer running SyncTunes directly:

```bash
git clone https://github.com/usaidt/synctunes.git
cd synctunes
go run ./cmd/server
```

## 🌍 Sharing with Friends Worldwide

Want friends outside your network to join? Use ngrok to create a public tunnel to your SyncTunes instance:

```bash
# Install ngrok from https://ngrok.com/download
ngrok http 8081
```

Ngrok will give you a public URL like `https://abc123.ngrok.io` that you can share with anyone. Your friends can join your listening rooms using this link, no matter where they are!

## 🎧 How It Works

**For Hosts:**
1. Start SyncTunes and add your music files to the `music/` folder
2. Create a room with any name you like
3. Share the room URL with friends (use ngrok for remote friends)
4. Browse your music collection and start playing tracks
5. Everyone in the room will hear the same music simultaneously

**For Listeners:**
1. Click the room link shared by your friend
2. Enter your name and join the room
3. Sit back and enjoy the synchronized music experience
4. See what's playing and who else is listening

## 🎵 Supported Music Formats

SyncTunes works with virtually any audio format your browser can play:

**Audio Files:** MP3, WAV, FLAC, OGG, M4A, and more
**Video Files:** MP4, MKV, AVI, MOV, WebM (audio-only playback)

Simply drag and drop your music files into the `music/` folder and they'll automatically appear in your catalog.

## ⚙️ Configuration

SyncTunes works out of the box, but you can customize it:

**Port:** Set `PORT=3000` environment variable to change from default port 8080
**Music Directory:** Set `MUSIC_DIR=/path/to/music` to use a different music folder

For Docker users, edit the `docker-compose.yml` file to mount your preferred music directory.

## 🛠️ Technical Details

SyncTunes is built with modern web technologies for reliability and performance:

**Backend:** Go with Gorilla WebSockets for real-time communication
**Frontend:** Alpine.js and HTMX for reactive interfaces without heavy frameworks
**Styling:** Tailwind CSS for a clean, responsive design
**Architecture:** RESTful API with WebSocket connections for live synchronization

The application manages room state, user connections, and playback synchronization automatically. When someone plays, pauses, or skips a track, all connected listeners receive the update instantly.

## Hackclub

I have used some AI for making the readme and for doing some golang coding becoz this is my first time using golang.