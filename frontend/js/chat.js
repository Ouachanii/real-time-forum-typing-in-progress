function initializeChatSystem(nickname = localStorag.getItem('nickname')) {    
    if (!nickname) return;
    
    // Multi-window
    const TAB_ID = `tab_${Date.now()}_${Math.random().toString(36).slice(2, 11)}`;
     // Listen for storage events from other tabs
     window.addEventListener('storage', (event) => {
        if (event.key === 'chat_message_update' && event.newValue) {
            const data = JSON.parse(event.newValue);  
            // Ignore messages from our own tab (case when we are in the same browser with different users loged and try to send a message from one to other)
            if (data.tabId !== TAB_ID) {
                if (data.type === 'new_message') {
                    displayPrivateMessage(data.message);
                    updateOnlineUsersList();
                }
            }
        }
        if (event.key === 'chat_notification_update' && event.newValue) {
            const data = JSON.parse(event.newValue);
            
            if (data.tabId !== TAB_ID) { // Ignore messages from our own tab (case when we are in the same browser with different users loged and try to send a message from one to other)

                // Update local state
                unreadCounts[data.sender] = data.unreadCount;
                userActivity[data.sender] = data.lastActivity;
                
                // Find the user element
                const userElement = document.querySelector(`.online-user[data-nickname="${data.sender}"]`);
                
                if (userElement) {
                    // Update badge
                    const existingBadges = userElement.querySelectorAll(".unread-badge");
                    existingBadges.forEach(badge => badge.remove());
                    
                    if (unreadCounts[data.sender] > 0) {
                        const badge = document.createElement("span");
                        badge.className = "unread-badge";
                        badge.textContent = unreadCounts[data.sender];
                        userElement.appendChild(badge);
                    }
                    
                    // Trigger UI update
                    updateOnlineUsersList();
                }
            }
        }
    });
 
    let socket = null;
    let allUsers = [];
    let onlineUsers = [];
    const userActivity = {}; // Tracks last message time
    const unreadCounts = {}; // Tracks unread messages per user
    let currentOpenChat = null;
    const typingTimers = {}; // Stores timeout IDs for sending 'typing_stop' per chat { chatNickname: timeoutId }
    const isTypingMap = {}; // Tracks if the current user is marked as typing in a chat { chatNickname: boolean }

    // Show loading state
    const userList = document.getElementById("onlineUserList");
    if (userList) {
        userList.innerHTML = '<li class="loading">Loading users...</li>';
    }
    
    // First fetch all users
    fetchAllUsers(nickname)
      .then(() => {
        // Then establish WebSocket connection
        initializeWebSocket(nickname);
      })
      .catch(error => {
        console.error('Error initializing chat:', error);
        const userList = document.getElementById("onlineUserList");
        if (userList) {
          userList.innerHTML = '<li class="error">Failed to load users. Please try again.</li>';
        }
      });
    
    // Define all the helper functions that were in your DOMContentLoaded
    function fetchAllUsers(nickname) {
      return new Promise((resolve, reject) => {
        fetch(`/get_all_users?nickname=${encodeURIComponent(nickname)}`)
          .then(response => {
            if (!response.ok) {
              throw new Error(`Server responded with ${response.status}`);
            }
            return response.json();
          })
          .then(users => {
            allUsers = users;
            resolve(users);
          })
          .catch(error => {
            console.error('Error fetching users:', error);
            reject(error);
          });
      });
    }
    
    function initializeWebSocket(nickname) {
      socket = new WebSocket(`ws://localhost:3344/ws?nickname=${nickname}`);

      // Fetch notifications when page loads
      fetch(`/get-notifications?nickname=${nickname}`)
      .then(response => response.json())
      .then(notifications => {
        if (notifications !== null){
            notifications.forEach(notif => {
                console.log("Unread from:", notif.sender);
            });
        }
      });

      socket.onopen = () => {
        console.log("Connected to WebSocket server");
        // Request online users explicitly after connection
        socket.send(JSON.stringify({
            type: "requestOnlineUsers"
        }));
    };
    
    socket.onclose = (event) => {
        console.log("Disconnected from WebSocket server", event.reason);
        
        if (event.reason === "User logged out") {
            // Visual feedback for offline status
            document.querySelectorAll('.status-dot').forEach(dot => {
              dot.classList.remove('online');
              dot.classList.add('offline');
              dot.title = 'Offline';
            });
          }
    };
    
    socket.onmessage = (event) => {
        const data = JSON.parse(event.data);
        switch (data.type) {
            case "userRegistered":
            const newUser = data.user;
            if (!allUsers.some(u => u.nickname === newUser.nickname)) {
                allUsers.push({
                    nickname: newUser.nickname,
                    firstName: newUser.firstName,
                    lastName: newUser.lastName,
                    isOnline: false
                });
                updateOnlineUsersList();
            }
            break;
          case "onlineUsers":
            onlineUsers = data.users;
            updateOnlineUsersList();
            break;
          case "notification":
            showNotification(data.sender);
            break;
          case "conversation_data":
            window.conversationData = data.data;
            updateOnlineUsersList();
            break;
          // --- Typing Indicator Handling ---
          case "typing_start": // Matches MessageTypeTypingStart in Go
            showTypingIndicator(data.sender, data.firstName); // Show indicator
            break;
          case "typing_stop": // Matches MessageTypeTypingStop in Go
            hideTypingIndicator(data.sender); // Hide indicator
            break;
          // --- End Typing Indicator Handling ---
          case "chat_message": // Matches MessageTypeChat in Go
             if (data.receiver) {
               // Hide typing indicator if it was shown for this sender
               hideTypingIndicator(data.sender);
               // Update lastActivity for actual messages
               userActivity[data.sender] = Date.now();
               displayPrivateMessage(data);
               updateOnlineUsersList();
               // Notify other tabs
               localStorage.setItem('chat_message_update', JSON.stringify({
                 tabId: TAB_ID,
                 type: 'new_message',
                 message: data
                 }));
                 localStorage.removeItem('chat_message_update'); // Clear the event
             }
             break;
          default: // Handle potential older message format or unknown types
            // Check if it looks like a chat message based on essential fields
            if (data.receiver && data.content && data.sender) {
              // Hide typing indicator if it was shown for this sender
              hideTypingIndicator(data.sender);
              // Update lastActivity for actual messages
              userActivity[data.sender] = Date.now();
              displayPrivateMessage(data);
              updateOnlineUsersList();
              // Notify other tabs
              localStorage.setItem('chat_message_update', JSON.stringify({
                tabId: TAB_ID,
                type: 'new_message',
                message: data
                }));
                localStorage.removeItem('chat_message_update'); // Clear the event
            }
        }
      };
    
    socket.onerror = (error) => {
        console.error("WebSocket error:", error);
    };
    }

    const showNotification = (sender) => {
        // Update unread count
        unreadCounts[sender] = (unreadCounts[sender] || 0) + 1;
        
        // Update last activity time
        userActivity[sender] = Date.now();
        
        // Find the user element
        const userElement = document.querySelector(`.online-user[data-nickname="${sender}"]`);
        
        if (userElement) {
            // Highlight the user
            userElement.style.backgroundColor = "#f1a564";
            userElement.style.transition = "background-color 0.3s ease";
            
            // Update badge
            const existingBadges = userElement.querySelectorAll(".unread-badge");
            existingBadges.forEach(badge => badge.remove());
            
            if (unreadCounts[sender] > 0) {
                const badge = document.createElement("span");
                badge.className = "unread-badge";
                badge.textContent = unreadCounts[sender];
                userElement.appendChild(badge);
            }
            
            setTimeout(() => {
                userElement.style.backgroundColor = "";
            }, 3000);
            
            // Trigger full list update instead of manual reordering
            updateOnlineUsersList();
        }

         // Notify other tabs about this notification
        localStorage.setItem('chat_notification_update', JSON.stringify({
            tabId: TAB_ID,  // Use the same TAB_ID from your initialization
            sender: sender,
            unreadCount: unreadCounts[sender],
            lastActivity: userActivity[sender]
        }));
        localStorage.removeItem('chat_notification_update'); // Clear the event
    };
    
    let isLoading = false;
    let allMessagesLoaded = false;
    let currentOffset = 0;

    function fetchHistoricalMessages(otherNickname, offset = 0, append = false) {
        const limit = 10;
        const messageList = document.getElementById(`messages-${otherNickname}`);
        if (!messageList) return;

        if (!append) {
            messageList.innerHTML = '<li class="loading-message">Loading messages...</li>';
            currentOffset = 0;
            allMessagesLoaded = false;
        } 

        else if (append && !isLoading && !allMessagesLoaded) {
            const loadingIndicator = document.createElement('li');
            loadingIndicator.className = 'loading-message';
            loadingIndicator.textContent = 'Loading more messages...';
            messageList.insertBefore(loadingIndicator, messageList.firstChild);
        }

        if (isLoading || allMessagesLoaded ) return;

        isLoading = true;

        fetch(`/fetch_messages?nickname=${encodeURIComponent(nickname)}&otherUser=${encodeURIComponent(otherNickname)}&offset=${offset}&limit=${limit}`)
        .then(response => response.json())
        .then(data => {     
            
            if ((Array.isArray(data) && data.length === 0)) {
                allMessagesLoaded = true;
                return;
            }

            const messages = Array.isArray(data) ? data : [];

            displayMessages(messages, messageList, append);

            currentOffset += messages.length;

            if (messages.length < limit) {
                allMessagesLoaded = true;
            }
        })
        .catch(error => {
            console.error('Error loading messages:', error);
        })
        .finally(() => {
            isLoading = false;
        });
    }

function displayMessages(messages, messageList, append) {
    // Remove loading indicator
    const loadingIndicator = messageList.querySelector('.loading-message');
    if (loadingIndicator) messageList.removeChild(loadingIndicator);

    // Sort messages chronologically
    messages.sort((a, b) => new Date(a.timestamp) - new Date(b.timestamp));

    // Create message elements using the new structure
    const messageElements = messages.map(msg => {
        const messageElement = document.createElement('li');
        messageElement.className = msg.sender === nickname ? "sent-message" : "received-message";

        const displayName = msg.sender === nickname
            ? "You"
            : `${msg.firstName || ''} ${msg.lastName || ''}`.trim() || 'Unknown';

        // Format timestamp (same logic as in displayPrivateMessage)
        let timeDisplay = msg.timestamp || 'No timestamp'; // Fallback
        try {
            const messageDate = new Date(msg.timestamp);
            if (!isNaN(messageDate.getTime())) {
                timeDisplay = formatTimeAgo(messageDate);
            } else {
                 console.warn("Could not parse historical timestamp for message:", msg);
            }
        } catch (e) {
            console.error("Error parsing historical timestamp:", e, msg.timestamp);
        }

        messageElement.innerHTML = `
            <img src="/frontend/assets/profile.png" alt="Avatar" class="message-avatar">
            <div class="message-content-wrapper">
                <div class="message-header">
                    <span class="message-author">${displayName}</span>
                    <span class="message-time">${timeDisplay}</span>
                </div>
                <p class="message-text">${EscapeString(msg.content || '')}</p> <!-- Ensure content is escaped -->
            </div>
        `;
        return messageElement;
    });

    if (append) {
        // Save scroll state before adding messages
        const scrollPos = messageList.scrollTop;
        const scrollHeight = messageList.scrollHeight;
        
        // Add messages to top in reverse order (oldest first)
        messageElements.reverse().forEach(msg => {
            messageList.insertBefore(msg, messageList.firstChild);
        });
        
        // Restore scroll position relative to new content
        messageList.scrollTop = scrollPos + (messageList.scrollHeight - scrollHeight);
    } else {
        // Initial load - add to bottom (newest first)
        messageList.innerHTML = '';
        messageElements.forEach(msg => {
            messageList.appendChild(msg);
        });
        messageList.scrollTop = messageList.scrollHeight;
    }
}

function setupScrollHandler(nickname) {
    const messageList = document.getElementById(`messages-${nickname}`);
    if (!messageList) return;
    
    let scrollDebounceTimer = null;
    
    messageList.addEventListener('scroll', () => {
        // Clear any pending debounce
        if (scrollDebounceTimer) {
            clearTimeout(scrollDebounceTimer);
        }
        
        // Set new debounce
        scrollDebounceTimer = setTimeout(() => {
            // Check if we're near top and should load more
            if (messageList.scrollTop < 50 && !isLoading && !allMessagesLoaded) {
                fetchHistoricalMessages(nickname, currentOffset, true);
            }
        }, 250);
    });
}
    
    
    window.openPrivateChat = async (nickname, firstName, lastName) => {

        const recieverOfNoti = localStorage.getItem("nickname") 
        try {
            await fetch('/mark-read', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    receiver: recieverOfNoti, 
                    sender: nickname              
                })
            });
    
            if (currentOpenChat) {
                window.closeChat(currentOpenChat);
            }
    
            const chatBox = document.getElementById(`chat-${nickname}`) || 
            createChatBox(nickname, firstName, lastName);
            
            chatBox.style.display = "block";
            currentOpenChat = nickname;
            
            fetchHistoricalMessages(nickname);
            setupScrollHandler(nickname);
            resetUnreadCount(nickname);
    
        } catch (error) {
            console.error("Chat opening failed:", error);
            showToast("Failed to open chat");
        }
    };
    
    const createChatBox = (nickname, firstName, lastName) => {
        const chatBox = document.createElement("div");
        chatBox.id = `chat-${nickname}`;
        chatBox.className = "private-chat";
        chatBox.innerHTML = `
            <div class="chat-header">
              <h4>Chat with ${firstName} ${lastName}</h4>
              <button class="close-chat">×</button>
            </div>
            <ul class="chat-messages" id="messages-${nickname}"></ul>
            <div class="typing-indicator" id="typing-${nickname}" style="display: none; padding: 0 10px; font-style: italic; color: var(--text-muted); height: 20px; line-height: 20px; font-size: 0.9em;"></div>
            <div class="chat-input-container">
              <input type="text" id="input-${nickname}" placeholder="Type a message...">
              <button class="send-message">Send</button>
            </div>
        `;
        
        const input = chatBox.querySelector(`#input-${nickname}`);
        const sendButton = chatBox.querySelector('.send-message');
        const closeButton = chatBox.querySelector('.close-chat');

        input.addEventListener('keypress', (event) => handleKeyPress(event, nickname));
        // Typing listeners added
        input.addEventListener('input', () => handleTyping(nickname));
        input.addEventListener('blur', () => handleStopTyping(nickname)); // Send stop on blur

        sendButton.addEventListener('click', () => sendPrivateMessage(nickname));
        closeButton.addEventListener('click', () => {
             handleStopTyping(nickname); // Ensure typing stops when chat is closed
             closeChat(nickname);
        });

        document.getElementById("chatContainer").appendChild(chatBox);
        return chatBox;
    };
    
    function handleKeyPress(event, nickname) {
        if (event.key === 'Enter') {
            sendPrivateMessage(nickname);
        }
    }


    
    window.sendPrivateMessage = (receiver) => {
        const messageInput = document.getElementById(`input-${receiver}`);
        const message = messageInput.value.trim();
        
        if (message && socket && socket.readyState === WebSocket.OPEN) {
            const data = {
                sender: nickname,
                receiver,
                content: message,
                timestamp: new Date().toLocaleTimeString(),
            };
            
            // Before sending the actual message, stop typing notification
            handleStopTyping(receiver);

            // Send the chat message (ensure type is set if backend expects it)
             const messageData = {
                 type: "chat_message", // Explicitly set type
                 sender: nickname,
                 receiver,
                 content: message,
                 // Timestamp is usually added by backend, but can be added here if needed
                 // timestamp: new Date().toISOString(), // Example ISO format
             };
            socket.send(JSON.stringify(messageData));

            messageInput.value = ""; // Clear input after sending

            // Display the sent message locally immediately
            displayPrivateMessage({
                ...messageData, // Use the sent data
                firstName: "You", // Display as "You" for sender
                lastName: "",
                // Use a client-side timestamp for immediate display if needed
                timestamp: new Date().toISOString()
            });

            // Update activity and UI (Corrected - remove duplicated block below)
            userActivity[receiver] = Date.now();
            updateOnlineUsersList(); // This will move the user to "Active Conversations" if needed
            resetUnreadCount(receiver);
        } else if (!message) {
            console.log("Empty message, not sending");
        } else {
            console.error("WebSocket not connected, cannot send message");
            alert("Connection lost. Please refresh the page to reconnect.");
        }
    };

    // --- Typing Indicator Logic ---

    function sendTypingNotification(receiver, type) {
        // Ensure type is either 'typing_start' or 'typing_stop'
        if (type !== 'typing_start' && type !== 'typing_stop') {
            console.error("Invalid typing notification type:", type);
            return;
        }
        if (socket && socket.readyState === WebSocket.OPEN) {
            const data = {
                type: type, // 'typing_start' or 'typing_stop'
                sender: nickname, // Current user's nickname
                receiver: receiver // The user being chatted with
            };
            socket.send(JSON.stringify(data));
        } else {
            console.error("WebSocket not connected, cannot send typing notification.");
        }
    }

    function handleTyping(receiverNickname) {
        // Clear existing timer if user continues typing
        if (typingTimers[receiverNickname]) {
            clearTimeout(typingTimers[receiverNickname]);
        }

        // Send 'typing_start' only if not already marked as typing
        if (!isTypingMap[receiverNickname]) {
            sendTypingNotification(receiverNickname, 'typing_start');
            isTypingMap[receiverNickname] = true;
        }

        // Set a timer to send 'typing_stop' after a delay (e.g., 3 seconds)
        typingTimers[receiverNickname] = setTimeout(() => {
            sendTypingNotification(receiverNickname, 'typing_stop');
            isTypingMap[receiverNickname] = false; // Mark as not typing
            delete typingTimers[receiverNickname]; // Clean up timer ID
        }, 1000); // 1 seconds delay
    }

    function handleStopTyping(receiverNickname) {
        // If there's an active timer, clear it
        if (typingTimers[receiverNickname]) {
            clearTimeout(typingTimers[receiverNickname]);
            delete typingTimers[receiverNickname];
        }
        // If the user was marked as typing, send 'typing_stop' immediately
        if (isTypingMap[receiverNickname]) {
            sendTypingNotification(receiverNickname, 'typing_stop');
            isTypingMap[receiverNickname] = false;
        }
    }

    function showTypingIndicator(senderNickname, senderFirstName) {
        // Chat window indicator
        const indicator = document.getElementById(`typing-${senderNickname}`);
        if (indicator) {
            const name = senderFirstName || senderNickname;
            indicator.textContent = `${EscapeString(name)} is typing`;
            indicator.style.display = 'block';
            indicator.style.color = 'green';
        }
        // User list indicator
        const userListIndicator = document.getElementById(`userlist-typing-${senderNickname}`);
        if (userListIndicator) {
            userListIndicator.textContent = 'typing...';
            userListIndicator.style.display = 'inline';
        }
    }

    function hideTypingIndicator(senderNickname) {
        // Chat window indicator
        const indicator = document.getElementById(`typing-${senderNickname}`);
        if (indicator) {
            indicator.textContent = '';
            indicator.style.display = 'none';
        }
        // User list indicator
        const userListIndicator = document.getElementById(`userlist-typing-${senderNickname}`);
        if (userListIndicator) {
            userListIndicator.textContent = '';
            userListIndicator.style.display = 'none';
        }
    }

    // --- End Typing Indicator Logic ---


    const updateOnlineUsersList = () => {
    // Combine online status with all users data
    const combinedUsers = allUsers.map(user => {
        const isOnline = onlineUsers.some(u => u.nickname === user.nickname);
        const lastActivity = userActivity[user.nickname] || 0;
        const unread = unreadCounts[user.nickname] || 0; 
        return {
            ...user,
            isOnline,
            lastActivity,
            unread
        };
    });

    // Check if we have any online users not in allUsers (newly registered)
    onlineUsers.forEach(onlineUser => {
      if (!allUsers.some(user => user.nickname === onlineUser.nickname)) {
        combinedUsers.push({
          nickname: onlineUser.nickname,
          firstName: onlineUser.firstName,
          lastName: onlineUser.lastName,
          isOnline: true,
          lastActivity: Date.now(),
          unread: 0
        });
      }
    });

    updateOnlineUsers(combinedUsers);
};

const updateOnlineUsers = (users) => {
    const userList = document.getElementById("onlineUserList");
    if (!userList) return;

    const convData = window.conversationData || { with_conversations: [] };
    const withConvs = new Set(convData.with_conversations || []);

    // Split users into groups
    const withConvUsers = users.filter(user => 
        user.nickname !== nickname && withConvs.has(user.nickname)
    );
    const withoutConvUsers = users.filter(user => 
        user.nickname !== nickname && !withConvs.has(user.nickname)
    );

    // Sort active conversations by last message time (not connection time)
    const sortedWithConv = withConvUsers.sort((a, b) => 
        (b.lastActivity || 0) - (a.lastActivity || 0)
    );

    // Sort other users by last activity (if any) then alphabetically
    const sortedWithoutConv = withoutConvUsers.sort((a, b) => {
        // If both have activity, sort by most recent
        if (a.lastActivity && b.lastActivity) {
            return b.lastActivity - a.lastActivity;
        }
        // If only one has activity, put that first
        if (a.lastActivity) return -1;
        if (b.lastActivity) return 1;
        // Otherwise sort alphabetically
        return a.firstName.localeCompare(b.firstName, undefined, { sensitivity: 'base' });
    });

    // Clear and rebuild list
    userList.innerHTML = '';

    // Add active conversations
    if (sortedWithConv.length > 0) {
        const header = document.createElement('li');
        header.className = 'section-header';
        header.textContent = 'Active Conversations';
        userList.appendChild(header);

        sortedWithConv.forEach(user => createUserElement(user));
    }

    // Add other users
    if (sortedWithoutConv.length > 0) {
        const header = document.createElement('li');
        header.className = 'section-header';
        header.textContent = sortedWithConv.length > 0 ? 'Other Users' : 'All Users';
        userList.appendChild(header);

        sortedWithoutConv.forEach(user => createUserElement(user));
    }
};

// Helper function to create consistent user list items
function createUserElement(user) {
    const userElement = document.createElement('li');
    userElement.className = 'online-user';
    userElement.dataset.nickname = user.nickname;

    // Profile Image
    const profileImg = document.createElement('img');
    profileImg.src = '/frontend/assets/profile.png';
    profileImg.alt = 'Profile';
    profileImg.className = 'user-list-avatar';

    // Status indicator
    const statusDot = document.createElement('span');
    statusDot.className = `status-dot ${user.isOnline ? 'online' : 'offline'}`;

    // Container for image and status dot
    const avatarContainer = document.createElement('div');
    avatarContainer.className = 'avatar-container';
    avatarContainer.appendChild(profileImg);
    avatarContainer.appendChild(statusDot);

    // Name display
    const nameContainer = document.createElement('div');
    nameContainer.className = 'user-name-container';
    nameContainer.innerHTML = `
        <span class="user-first-name">${user.firstName}</span>
        <span class="user-last-name">${user.lastName}</span>
    `;

    // Typing indicator for user list
    const typingIndicator = document.createElement('span');
    typingIndicator.className = 'user-list-typing-indicator';
    typingIndicator.id = `userlist-typing-${user.nickname}`;
    typingIndicator.style.marginLeft = '8px';
    typingIndicator.style.color = 'green';
    typingIndicator.style.fontStyle = 'italic';
    typingIndicator.style.display = 'none';

    // Unread badge
    if (user.unread > 0) {
        const badge = document.createElement('span');
        badge.className = 'unread-badge';
        badge.textContent = user.unread;
        userElement.appendChild(badge);
    }

    // Append avatar, name, and typing indicator
    userElement.append(avatarContainer, nameContainer, typingIndicator);
    userElement.onclick = () => {
        openPrivateChat(user.nickname, user.firstName, user.lastName);
        resetUnreadCount(user.nickname);
    };

    document.getElementById("onlineUserList").appendChild(userElement);
}
    

function EscapeString(unsafeStr) {
    return unsafeStr
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;")
        .replace(/'/g, "&#039;");
}

const displayPrivateMessage = (data) => {
    if (!data || !data.sender || !data.receiver) {
        console.warn('Invalid message data received');
        return;
    }

    data.content = EscapeString(data.content)


    const chatWith = data.sender === nickname ? data.receiver : data.sender;
    const messageList = document.getElementById(`messages-${chatWith}`);
    
    // Update the last activity time for this user
    if (chatWith) {
        userActivity[chatWith] = Date.now();
    }

    // Add the message to the chat if it exists
    if (messageList) {
        const displayName = data.sender === nickname 
            ? "You"
            : `${data.firstName || ''} ${data.lastName || ''}`.trim() || 'Unknown';
        
        // Attempt to parse the timestamp for formatTimeAgo
        let timeDisplay = data.timestamp || 'No timestamp'; // Fallback
        try {
            const messageDate = new Date(data.timestamp);
            // Check if the date is valid before formatting
            if (!isNaN(messageDate.getTime())) {
                timeDisplay = formatTimeAgo(messageDate);
            } else {
                 console.warn("Could not parse timestamp for message:", data);
            }
        } catch (e) {
            console.error("Error parsing timestamp:", e, data.timestamp);
        }

        const messageElement = document.createElement('li');
        messageElement.className = data.sender === nickname ? "sent-message" : "received-message";
        
        messageElement.innerHTML = `
            <img src="/frontend/assets/profile.png" alt="Avatar" class="message-avatar">
            <div class="message-content-wrapper">
                <div class="message-header">
                    <span class="message-author">${displayName}</span>
                    <span class="message-time">${timeDisplay}</span>
                </div>
                <p class="message-text">${data.content || ''}</p>
            </div>
        `;
        
        messageList.appendChild(messageElement);
        messageList.scrollTop = messageList.scrollHeight; // Scroll to bottom
    }

    // Initialize conversation data if it doesn't exist
    window.conversationData = window.conversationData || { with_conversations: [] };
    
    // Safely access the conversations array
    const conversations = window.conversationData.with_conversations || [];
    
    // Add user to conversations if not already there
    if (chatWith && !conversations.includes(chatWith)) {
        conversations.push(chatWith);
        window.conversationData.with_conversations = conversations;
        
        // Notify server about the new conversation
        if (socket?.readyState === WebSocket.OPEN) {
            socket.send(JSON.stringify({
                type: "update_conversations",
                with_conversations: conversations
            }));
        }
    }

    // Always update the UI to reflect recent activity
    updateOnlineUsersList();
};
    
    window.closeChat = (nickname) => {
        const chatBox = document.getElementById(`chat-${nickname}`);
        if (chatBox) chatBox.style.display = "none";
        currentOpenChat = null; // Reset the tracker
    };
    
    const resetUnreadCount = (nickname) => {
        unreadCounts[nickname] = 0; // Set to 0 instead of deleting to maintain the key
        const userElement = document.querySelector(`.online-user[data-nickname="${nickname}"]`);
        if (userElement) {
            const badge = userElement.querySelector(".unread-badge");
            if (badge) badge.remove();
        }
    };
    
    // Poll for user status updates every 30 seconds
    const userStatusInterval = setInterval(() => {
        if (socket && socket.readyState === WebSocket.OPEN) {
            socket.send(JSON.stringify({
                type: "requestOnlineUsers"
            }));
        }
    }, 30000);
    
    // Clean up interval on page unload
    window.addEventListener('beforeunload', () => {
        clearInterval(userStatusInterval);
    });
  }

// this part used when the page reloaded to save the list of users ... 
document.addEventListener("DOMContentLoaded", () => {
    const nickname = localStorage.getItem("nickname");
    if (nickname) {
        initializeChatSystem(nickname);
        
        // Show chat interface and hide login
        document.getElementById("loginContainer").style.display = "none";
        document.getElementById("chatContainer").style.display = "block";
    }
});

// to export the function and can call it from any other script
window.initializeChatSystem = initializeChatSystem;
