# Overview

High level: KiKi is a distributed social network that you can use off the grid, which provides secure end-to-end communication with people inside your social circles.

Medium level: Kiki is like a poor-man's mailservice. Every user in Kiki is a mail carrier. They carry mail for other people and receive mail from other people. Letters are sealed using special encryption that only allows the recipient (friends, public) to read them.

Low level: All the information in the network is stored in **Letters** that are sealed in special cryptographically secure **Envelopes**. In KiKi, you - the **Person** - are essentially a pair of keys used for [asymetrical cryptography](https://en.wikipedia.org/wiki/Public-key_cryptography). These keys are used to open Envelopes and seal and sign Letters. You interact with with KiKi through a **Feed** that displays the contents of all Letters that you can open. These letters are downloaded to your computer, so you can access the Feed anytime, even off-line.

Everyone on KiKi also carries Letters addressed to others. This makes KiKi distributed, because everyone is a mailman/mail-woman. Once two people encounter each other on LAN or when connecting to a public server, they will exchange Letters that they carried for each other. This ensures that the network can exist as a mesh, outside of the realm of ISPs. It also ensures that no federated servers are necessary, it will work with a few people that can connect onto a local network.

## Ideas

Assignment allows assesment. Reputation can be evaluated through arbitrary quantification of certain public aspects of the social network. I.e. - posts that are more popular have a certain amount of "likes", more reputable people have more followers, etc.

Contributions create connections. Kiki should be built around providing a frictionless path towards contributing content to be shared, locally, across the network.

## Access

Each social network has a different way of answering the question: *who can access what?* 

In KiKi the access to content is maintained through *four* levels of privacy. These privacy levels are based on whether you and another person are "friends" where "friends" which are two people that mutually agree to share information. 

These levels of privacy are maintained by using public key cryptography. Whenever you transmit information on KiKi, your information will be encrypted by a one or several of your keys, depending on the level of privacy. Here are the privacy levels, in order from private to public: 

1. **Personal**: only you can read (your Personal public key is used).
2. **Special**: you specify which friends can read (your Personal public key + Personal public keys of specified people is used).
3. **Friends**: all your friends can read (your Personal public keys + your Friends public keys is used).
4. **Region**: everyone can read (your Personal public key is used and the Region public key is used).

At a minimum, each user has three key pairs - a *Personal* key pair, a *Friends* and a *Region* key pair. Your Personal and Friends key pairs are unique to you. The Personal key pair identifies you as you, and is used to verify your Letters. The Friends key pair is transmitted to new friends which allows them to open Letters addressed to friends. The Region key pair is built-in to the application. It basically identifies a specific region (think State, City, Nation) in which communication is passed. A person can belong to multiple regions, and everyone must belong to at least one. 



### Personal key pair

Every time you use KiKi, you will seal your posts with your Personal private key. This prevents others from trying to write posts as you and enables you to send private messages to yourself (as in a diary) or to others.

You can have only one Personal key. If you have two computers and need to merge the accounts, you can simply copy the Personal keys from one computer to the other. All other information is stored in Letter which can be aquired by syncing with anyone.

### Friends key pair

Your first Friends key pair is generated when you start KiKi for the first time. It is emitted as a Letter to yourself, so it automatically syncs between machines.

When you follow someone else and that person also follows you, then you become "friends", and you send that friend the public and private keys for all your Friends key pairs. These keys allow your new friend to view all the posts you have made available to friends.

Unfriending happens. When you unfriend someone by unfollowing them, then you will generate a new Friends key pair. This new Friends key pair will be sent to the remaining friends and will not be sent to the unfriended person. This means that the unfriended person can not read your future messages, but they can still open the past messages (just like in real-life we still retain the memories with that friend up until they leave).

### Region key pair

Every instance of KiKi is configured with a Region key (or several). When you start KiKi it will be able to open any Envelope that it encounters that is meant for any one of your Regions. This allows sub-networks to be formed easily within KiKi, so that you can specify specific places like "United States" or "University of Alberta" to carry around the Envelopes.

## Reading 

The main interaction with KiKi is through reading personal Letters and Letters from friends in **Feed** (for more info [see structure of Feed](#feed)). The feed shows only posts that you are accessible to you (i.e. any Letters you can open with your available keys). The feed is generated from all of the opened Letters (see below), sorted by the time/category.

When you start KiKi, the feed is loaded from a locally stored database. KiKi will then check your LAN and registered servers for other KiKi instances. When another KiKi instance is detected, the two computers will sync up their lists of Letters. If these lists differ, then the computer will download each one it is missing, one-by-one, and try to open the Envelope to retrieve the Letter. The computer will use all available keys to try to open it (i.e. the Personal private keys and also the collected Friends private keys). If successfully opened, the Envelope is saved to the disk. Saving onto disk ensures that the feed will be accessible even if you are off-the-grid.

In order to make the network resilient, your computer will also download some Envelopes that cannot be opened by the keys that you hold. Since these can't be opened they will not be available in your feed. These will be stored on your computer, in the possibility that you might meet someone who needs them who will request them from your computer. In this way, you also act as a mailman/mailwoman who is carrying letters for other people. (You can set the limit the amount of space used for storing other people's envelopes by modifying `PublicEnvelopedDiskSpace=20MB` in the configuration file. The Envelopes that you can open do not count towards this limit.)

## Structure

KiKi has four main components which revolve around the basic functionality: You, the **Person**, can send a **Letter** which is sealed in an **Envelope**. Anyone who can unseal your Envelope to open the Letter will add it to their **Feed** to be able to read it, like it, or reply to it.

KiKi is written in Go. These is a basic overview of the `struct` classes used for storing the structures.

### Person

You, the **Person**, are just a pair of keys, the `Personal` key pair.

```golang
// Person is just a set of keys
type Person struct {
    Keys *keypair.KeyPair `json:"keys"`
}
```

The `Personal` key pair is stores on your computer in `$HOME/.kiki/identity.json`. The `Personal` key pair is for signing messages and encrypting messages that are from you. If you are using multiple computers, make sure to copy this to each machine to ensure you maintain your identity.

### Feed

When kiki loads, it is mainly used to display the *Feed*. The Feed contains all the information that is determined from opening Envelopes and reading Letters. The *Feed* is generated by

1. Synchronization with other available carriers
2. Opening all Envelopes
3. Lifting all keys from the Unsealed Envelopes
4. Goto #2 until no more new keys are discovered
5. Open remaining Envelopes
6. Use Unsealed Envelopes to discover information (posts, likes, etc.)

Feeds can be completlely reconstructed from the main database, `$HOME/.kiki/kiki.db`. One of the main objects is the `Feed` object, which is stored in the `Keystore` bucket, marshaled into the `feed` key. The `KeysForFriends` is a list of each public/private key given out to friends. Only the first key is used to send keys to friends. When a friend is blocked, then a new key is *prepended* to this list and that new key is posted to all the remaining friends. The friend that is blocked will not obtain this new key, therefore they will not be able to view any future messages. The `KeysFromFriends` is the list of public/private keys that can be used to unseal envelopes. If any of these are successful then you know the message is from a friend.

There are also some things that warrant their own bucket. The `AssignedNames` bucket contains a map of the string of the public key to the display name assigned by that person. `AssignedPhoto` is similar, it contains the public key of a user mapped to the base64 string of the image assigned by that public key.


### Envelope

The **Envelope** is the public meta data for a sealed **Letter** (see [Letter](#Letter), below) that tells users who the message is from and where it is going and contains the sealed (encrypted) data. 

```golang
// Envelope is the sealed letter to be transfered among carriers
type Envelope struct {
    Sender     *keypair.KeyPair `json:"sender"`     // public key of the sender
	Recipients []string         `json:"recipients"` // secret passphrase to open SealedContent,
	// encrypted by each recipient public key
	SealedContent string    `json:"sealed_content"` // encrypted compressed Letter
	Timestamp     time.Time `json:"timestamp"`      // time of entry
	ID            string    `json:"id"`             // hash of SealedContent
}
```

The `ID` is a SHA-256 sum of the `Data` of the LetterContent contained in the Letter. The `SealedContent` contains the marshaled LetterContent encrypted by a unique and random *secret passphrase*. The `Timestamp` records when the Envelope was sent. The `Recipients` is an array the *secret passphrase* encrypted by the public keys of each recipient. 

Each Letter is sealed in an Envelope which requires opening. To open you will need to decrypt the *secret passphrase* which will in turn decrypt the `SealedContent`. To decrypt the *secret passphrase*, you will try your private keys (e.g. your Personal key, your Region key, and all of your acquired Friends keys) against each element in the `Recipients`. If one of the recipients includes you, then one of your private key swill be able to decrypt the *secret passphrase* for unsealing the `SealedContent`. At scale (millions of users), a typical user might require using ~hundreds of private keys against ~tens of recipient ciphers, resulting in thousands of attempts per Letter. For typical computers, this will still only take 50 - 200 ms. (<small>IDEA: Add random amounts of "fake" recipients to obfuscate how many recipients there actually are</small>).

All the Envelopes are stored in a [bbolt database](https://github.com/asdine/storm) in `$HOME/.kiki/kiki.db`. This is the file that is synced when connecting to other KiKi instances. The primary key of this database is the ID, which has a unique restraint. Thus, to sync db X with db Y, you will simply try to add each row in X into Y.

When an Envelope is opened it reutns a `UnsealedEnvelope`:

```golang
// UnsealedEnvelope is created when an enveloped is opened
type UnsealedEnvelope struct {
    Sender     *keypair.KeyPair   `json:"sender"`    // public key of the sender
    Recipients []*keypair.Keypair `json:"recipients` // public key of the determined recipient
    Letter     *letter.Letter     `json:"letter"`    // the unsealed contents of the letter
    Timestamp  time.Time          `json:"timestamp"` // time of entry
    ID         string             `json:"id"`        // hash of SealedContent
}
```

The unsealed envelope is stored in a bbolt database in `$HOME/.kiki/feed.db` in a bucket `UnsealdEnvelopes`.


### Letter

The **Letter** is the sealed contents of the envelope, which contains the content of the message and some meta data about how it should be parsed in the feed.

```golang
// Letter contains meta data describing the content
type Letter struct {
	LatestID string        `json:"latest_id"` // hash of sender + un-encrypted data
	ID       string        `json:"id"`        // original ID, different than LatestID if overwriting
	Hashtags []string      `json:"channels"`  // channels for showing the post
	ReplyTo  string        `json:"reply_to"`  // hash that Letter is response to
	Content  LetterContent `json:"content"`
}

// LetterContent is the actual content of the letter
type LetterContent struct {
	Kind string `json:"kind"` // kind of letter content
	Data string `json:"data"` // base64 encoded bytes of data
}
```

All letters invoke actions. Typically these actions are just to post text/image to the designated recipients. However, there are other actions that help discern who is who and help to transfer identifies and key pairs between people.

All the Letters are stored in a [bbolt database](https://github.com/asdine/storm) in `$HOME/.kiki/letters.db`. This file contains unsealed Envelopes, so it will never be shared. To further ensure privacy you can enable `StoreLettersInMemory=true` in the configuration file to keep the database in memory (and rebuild on each startup). My [tests](https://gist.github.com/schollz/f08282396a8b184e30dddbe2422ba88a) determine that loading data from files in a database is about 70x faster than find and loading files from a file system.

Letters can have up to 3 "Hashtags". Hashtags are hashtags that indicate the group that this message should belong. Hashtags are determined by the last three hashtags by [using Regex](https://regex101.com/r/voKb2s/1).

#### Assigning name / profile / profile picture

When starting for the first time, you assign yourself a name (initially your name is just your public key). These Letters are published for the Region and have the following LetterContent:


```json
{
    "Kind":"assign-name",
    "Data":"Zack"
}
```

This Letter is signed by the Public key for Zack, so it ensures that this is what Zack wants to be called.

You can also assign profiles and pictures by changing the "Type" to "Profile" or "Picture", respectively.

## Posting a message

For a regular post, the LetterContent should look like:

```json
{
    "Kind":"post",
    "Data":"This is my first <em>post</em>"
}
```

Here the `Content` is simply the HTML of the post. There is some server processing done on this data. The server will find all base64 encoded images and re-encode them as JPG/PNG and submit the actual image as a new message:

```json
{
    "Kind":".jpg",
    "Data":"...base64 encoded data of JPG image..."
}
```

The server will server these as the `ID+Kind` where the `Kind` is just the extension.

The server will also find all the Hashtags by finding the last three tags [using Regex](https://regex101.com/r/voKb2s/1).

#### Following

When you follow someone, you emit a Letter that contains the public key of the person you are following. The Letter is sealed for only you and the person you are following, so you can read the feed to know who *you follow* and the other person can know that you are following them, since it is signed by your key.

```json
{
    "Kind":"follow",
    "Data":"nX2pwIjogIuOBk-OCk-OBq-OBoeOBr"
}
```

In this case the `Content` (`nX2pwIjogIuOBk-OCk-OBq-OBoeOBr`) is the public key of the person being followed. The person doing the following is determined by the public key of the sender (published in the envelope).


#### Sending a friend a key

You can send a Friends key to a friend. This happens automatically when two people follow each other.

```json
{
    "Kind":"keys",
    "Data":"JSON-encoded Friends key pair"
}
```

These letters are encrypted for a specific user, to ensure that 
the keys are transfered safely.

#### Emote

You can emote on people's posts and pictures. These are also Region messages (everyone in the region can see them) which have the following criteria:

```json
{
    "Kind":"like",
    "Data":"174d7c78..."
}
```

In this case, the `Content` (`174d7c78...`) is the SHA-256 sum of the post that is being assigned the like.

## Synchronization


Every user is a carrier. Every carrier is a server that can be connected to for file synchronization. Anytime kiki sees another kiki instance it attempts to synchronize as follows.

1. Get a list of files and the Region Key from other (`GET /catalog`).
2. Check if Region key matches, if so, continue.
3. Compare other's list to your list and find which items you do not have.
4. Get each item that you *do *not have* and that the other *does have* (`GET /envelope/X`) and insert into the database after verifying the signature.
5. Post each time that you *do have* and the other *does not have* (`POST /envelope`). Note that, when a server recieves a `POST /envelope` it will verify the signature before saving it.

Blocking someone should purge their messages and disallow syncing. When syncing, send a list of blocked public keys and then the other servers will remove those.

When making a release add the my public key encrypted by my private key and ensure they are same before doing any synchronization. Synchronize by just sending sql dumps?

Check size of a user on disk by adding up length of each column when querying all their rows, to make sure synchornization doesn't overflow the specified amount of data to save per person.

### Storage

Without bars, the carrier will ask for and hold *all envelopes* from *everybody*. This is not practical from a user-standpoint, so there should be some bars for posting things. 

One kind of limitation is to only hold a specified amount of data for each user - say 10MB for friends and 5MB for the public. Once this limitation is reached, then older items would be purged until the limit is no longer exceeded. 

This will prevent spam because someone posting hundreds of 1MB images will only get a few of them onto each computer before they reach their limit. This will also increase efficiency because it will allow computers to sync newer files, and stop synchronization once the limits are exceeded.



## Web Interface

The web interface contains all the opened envelopes served from memory.

## Security

TODO

## Comparison with existing systems

There are some other [similar-minded networks out there](https://github.com/topics/social-network):

- mastodon: federated microblogging (<=240 characters)
- diaspora:
- humhub:
- scuttlebutt:

Scuttlebutt is an append-only log, that does not allow editing / deleting posts. In *kiki* you can both edit and delete posts through the `Replace` ID of a Envelope.

# Roadmap

There are a number of facets that need work:

## Low level targets

- [ ] Public key loading/parsing
- [ ] Encryption / decryption
- [ ] Message bundling (envelopes + letters)
- [ ] Database operation (bboltdb, keystore, feed objects)

## Medium level targets

- [ ] Carrier server, implementation begun here: https://gist.github.com/schollz/f25d77afc9130b72390748bdbce0d9a3
- [ ] Command-line read/post (kiki read / kiki post something.md)

## High level targets

- [ ] Interactive web-based UI, implementation begun here: https://github.com/schollz/kiki/tree/master/kikiscratch

