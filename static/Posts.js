
var Post = function(data, api) {
    this.data = data;
    this.hash;
    this.api = api;
    this.comments = null;
    this.owner = null;
    this.callbacks = [];
    this.fetchComments();
    this.fetchOwner();
    this.$el = this.buildUi();
}

Post.prototype.buildUi = function() {
    return $("<div>");
}

Post.prototype.registerUpdateCallback = function(callback) {
    if ("function" == typeof(callback)) {
        this.callbacks.push(callback);
    }
}

Post.prototype.getContent = function() {
    return this.data.content;
}

Post.prototype.getPostId = function() {
    return this.data.id;
}

Post.prototype.getOwnerId = function() {
    return this.data.owner_id;
}

Post.prototype.fetchOwner = function(callback) {
    var self = this;
    if (this.owner) {
        callback && callback(null, this.owner);
        return;
    }
    app.Users.fetchUser(this.getOwnerId(), function(err, user){
        self.owner = user;
        callback && callback(err, user);
    });
}

Post.prototype.getRecipients = function() {
    return this.data.recipients;
}

Post.prototype.onUpdate = function() {
    for (var i=0; i<this.callbacks.length; i++) {
        this.callbacks[i]();
    }
}

Post.prototype.edit = function(content) {
    var letter = {
        "purpose": this.data.purpose,
        "replaces": this.getPostId(),
        "reply_to": this.data.reply_to,
        "to": ["self"],
        "content": content
    }
    this.api.submitLetter(letter, function(err, res) {
        console.log(err, res);
    });
}

Post.prototype.update = function() {
    var self = this;
    if (this.api) {
        this.api.fetchPost(this.getPostId(), function(err, res){
            if (err) throw err;
            self.data = res.data.post[0];
            self.onUpdate();
        });
    }
}

Post.prototype.fetchComments = function() {
    var self = this;
    if (0 < this.data.num_comments) {
        app.Posts.fetchPostComments(this.getPostId(), function(err, posts) {
            self.comments = posts;
        });
    } else {
        this.comments = [];
    }
}


var PostsCollection = function(api) {
    this.api = api;
    this.data = {};
    this.callbacks = {};
}

PostsCollection.prototype.addPost = function(data) {
    var post = new Post(data, this.api);
    this.data[data.id] = post;
    return post;
}

PostsCollection.prototype.addPosts = function(data) {
    var posts = [];
    for (var i=0; i<data.length; i++) {
        posts.push(this.addPost(data[i]));
    }
    return posts;
}

PostsCollection.prototype.fetchPosts = function(callback){
    var self = this;
    this.api.fetchPosts(function(err, res){
        var posts = [];
        if(!err && res.data.posts) {
            posts = self.addPosts(res.data.posts);
        }
        callback && callback(err, posts);
    });
}

PostsCollection.prototype.fetchPost = function(post_id, callback) {
    var self = this;
    if (this.data[post_id]) {
        return callback(null, this.data[post_id]);
    }
    if (!this.callbacks[post_id]) {
        this.callbacks[post_id] = [];
    }
    this.callbacks[post_id].push(callback);
    // only fire one request
    if (1 < this.callbacks[post_id].length){
        return
    };
    // fetch from api
    this.api.fetchPost(post_id, function(err, res){
        if (err) {
            self.runCallbacks(err, post_id);
            return;
        }
        if (res.data.posts && res.data.posts[0]) {
            self.data[post_id] = new Post(res.data.posts[0], self.api);
            self.runCallbacks(null, post_id);
        } else {
            self.runCallbacks(new Error("Not found"));
        }
    });
}

PostsCollection.prototype.fetchPostComments = function(post_id, callback) {
    var self = this;
    if (!this.data[post_id]) {
        return this.fetchPost(post_id, function(){
            self.fetchPostComments(post_id, callback);
        });
    }
    if (this.data[post_id] && this.data[post_id].comments) {
        return callback(null, this.data[post_id].comments);
    }
    if (!this.callbacks['comments_'+post_id]) {
        this.callbacks['comments_'+post_id] = [];
    }
    this.callbacks['comments_'+post_id].push(callback);
    // only fire one request
    if (1 < this.callbacks['comments_'+post_id].length){
        return
    };
    // fetch from api
    this.api.fetchPostComments(post_id, function(err, res){
        if (err) {
            self.runCommentCallbacks(err, post_id);
            return;
        }
        if (res.data.posts) {
            self.data[post_id].comments = self.addPosts(res.data.posts);
        } else {
            self.data[post_id].comments = [];
        }
        self.runCommentCallbacks(null, post_id);
    });

}

PostsCollection.prototype.runCommentCallbacks = function(err, post_id) {
    while (this.callbacks['comments_'+post_id].length) {
        var callback = this.callbacks['comments_'+post_id].shift();
        callback && callback(err, this.data[post_id].comments);
    }
    if (0 == this.callbacks['comments_'+post_id].length) {
        delete this.callbacks['comments_'+post_id];
    }
}

PostsCollection.prototype.runCallbacks = function(err, post_id) {
    while (this.callbacks[post_id].length) {
        var callback = this.callbacks[post_id].shift();
        callback && callback(err, this.data[post_id]);
    }
    if (0 == this.callbacks[post_id].length) {
        delete this.callbacks[post_id];
    }
}
