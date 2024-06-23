# Storage

This is where all the data storage logic is implemented.
This module does not take opinions on *when* an operation should occur, only _how_ it will be done.
That means other modules like the engine/handlers/servers must implement their authN/authZ logic.
The design, ideally, separates the concerns of how data is remembered from those that need to record/remember it.
