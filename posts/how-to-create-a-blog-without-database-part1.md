{"title": "How to create a blog without database — part1", "created": "2022-11-13", "tags": ["ruby", "ror", "redis"]}

# How to create a blog without database — part1

It’s been a while since the latest rails version was released, I am a little bit curious about the experience of building a dead simple blog using rails 7 in late 2022. It will be a lot of fun of building a straightforward blog system by myself.

## The tech stack

I am going to use rails 7 + Redis + tailwind CSS to implement the  system. Since I will deploy it in a very low end virtual machine, I try to make everything as simple as possible. I’d like to write the post via markdown language and save it to a plain text file, in that case, data persistence is optional. I still need to store the post list information, such as the post title, published date and post tags. The ideal way of saving this information is using a traditional relational database, for sample mysql or postgres, remind you that I’ve confirmed that avoiding using a database, I decided to use redis instead. So here is the design, every time I create a new post, I commit the file to a git repository , run a deploy script which will make the repository in vm updated，and in the end, I will run a rake task that pareses post information and store the data to redis. I will not add an expiration time when saving something to redis, in that way we can treat the cache as a straightforward mini database. 

- Ruby on Rails: the primary web framework
- Redis: temporary storage
- Tailwind css: css framework

## Design

I divide the entire progress into 2 phases, the parse phase and the render phase.

- parse phase: parse the markdown files, get the metadata of the post, store the data to redis
- render phase: when users view post list and the post detail, read the data from redis, if the cache expires, read the data from md file directly

A typical post file looks like this

```bash
{"title": "How to create a post", "created": "2022-10-17", "tags": ["python", "request"]}

# How to create a post

This is the content of the post.

```

The first line is a straightforward json string. Several fielders are mandatory.

- title: post title
- created: post created date

Tags field is optional.A tag is kind of a category, it aggregates similar posts.

## Routes

We only need to concern 2 pages.

- Post list: display all the posts in order by updated time from newer to older
- Post detail: the content of the post

## Create the project

At this moment, the latest rails version is 7.0.4, ideally we’d better use the latest ruby version, however ruby 3.1.x is sufficient , so I will stick to ruby 3.1.1.

```bash
# install rails 7.0.4
gem install rails -v 7.0.4
```

Create a new project using the rails command and indicate tailwindcss as the default css framework.

```bash
rails new mylog --css tailwind
```

Despite we are not going to use the database, the rails project still needs to init it.

```bash
cd mylog

rails db:create
```

Run rails server.

```bash

bin/dev
```

Now open the browser and go to localhost:3000, you can see the default rails welcome page.

## Create a redis config file

As I use redis as the primary storage and rails do not have a redis configuration file by default, it is time to create a redis config file manually.

```bash
touch config/redis.yml
```

Set some basic config, for example host and port.

```yaml
# config/redis.yml
production:
  host: localhost
  port: 6379
  
development:
  host: localhost
  port: 6379
  
```

## Create a rake task to parse markdown file

Parsing the markdown file is significant and a little bit complicated, let’s design the primary data structure first.

**post list set**

A redis zset contains all the posts information, the score is the post updated date, value is a json string that looks like the following

```json
1) "{\"title\":\"python requests\xe7\x9a\x84\xe6\x9b\xbf\xe4\xbb\xa3\xe8\x80\x85\xef\xbc\x9fhttpx\xe5\x88\x9d\xe4\xbd\x93\xe9\xaa\x8c\",\"created\":\"2022-10-17\",\"tags\":[\"python\",\"request\"],\"file_name\":\"first-post\"}"
2) "{\"title\":\"\xe7\xae\x97\xe6\xb3\x95\xe9\xa2\x98\",\"created\":\"2022-10-27\",\"tags\":[\"alg\",\"python\",\"string\"],\"file_name\":\"second-post\"}"
```

This data structure contains everything I need to display on the post list page which is also my homepage.

**tag list set**

I’d like to create a redis set to store all the tags since I have a fancy tag list on the homepage. Set is the ideal data structure because it guarantees that no duplicated items exist.

```jsx
127.0.0.1:6379> smembers dev_tags
1) "alg"
2) "string"
3) "request"
4) "python"
```

individual **tag  set**

As I have a tag list displayed on the homepage consequently end users can click the tag link and I’d like to navigate to a tag page that aggregates all the posts that belong to a certain tag.

That is to say I have to store all the posts with the same tag attached to them. This can be achieved by using some redis sets, the key of which contains a prefix and the tag name, the value is a json string which is a serialization of the post’s basic data such as title and markdown file name.

It looks just like this

```jsx
smembers dev_t_python
1) "{\"title\":\"python requests\xe7\x9a\x84\xe6\x9b\xbf\xe4\xbb\xa3\xe8\x80\x85\xef\xbc\x9fhttpx\xe5\x88\x9d\xe4\xbd\x93\xe9\xaa\x8c\",\"created\":\"2022-10-17\",\"tags\":[\"python\",\"request\"],\"file_name\":\"first-post\"}"
2) "{\"title\":\"\xe7\xae\x97\xe6\xb3\x95\xe9\xa2\x98\",\"created\":\"2022-10-27\",\"tags\":[\"alg\",\"python\",\"string\"],\"file_name\":\"second-post\"}"
```

In the example, `dev_t_`is the prefix, python is the tag name. There are two items in the set, each of them is a json string that includes post’s metadata.

**post detail string** 

In the end we have to save the post content which will fulfill the post detail page to redis. This is straightforward, we can combine a prefix with the post’s filename to generate a unique key and store the markdown text to redis.

### create a brand new rake task

Type the following commands in the terminal.

```bash
rails g task scan
mkdir -p posts 
touch posts/first_post.md
touch posts/second_post.md
```

Open scan.rake file that locates in lib/task directory, below is the complete code.

```ruby
namespace :scan do

	desc 'parse file'
  task :parse => :environment do
		r = Redis.new(host: Rails.configuration.redis['host'], port: Rails.configuration.redis['port'])
		key = Rails.configuration.redis['post_set_key']
    set_key = Rails.configuration.redis['tag_set_key']
    r.del(key)
    r.del(set_key)
    clear_all_tags(r)

		Dir[File.join(Rails.root, "posts", '*.md')].each do |file|
			file_name = file.split(File::SEPARATOR)[-1].sub('.md', '')
      md_content = []
      meta = {}
      File.readlines(file).each_with_index do |line, index|
        if index == 0
          meta = parse_meta(line)
          next if meta.blank? 
        else
          md_content.push line 
        end #if
      end
      unless meta.blank?
        puts "procssing #{file_name}"
        meta['file_name'] = file_name
        save_content_to_redis(r, file_name, md_content.join(), meta['tags'])
        parse_tags_and_save_to_redis(r, meta)
        r.zadd(key, File.mtime(file).to_datetime.to_i, JSON.dump(meta))
      end #unless
    end #each
  end #task

  def parse_meta(meta_str)
    required_fields = %w{title created}

    res = JSON.load(meta_str) rescue false
    if res 
      required_fields.each do |field|      
        if not res.key?(field)
          return {} 
        end #if
      end #each
    else
      return {}
    end #if 
    res
  end

  def parse_tags_and_save_to_redis(r, meta)
    puts "Parsing tags"
    set_key = Rails.configuration.redis['tag_set_key']
    meta['tags'].each do |tag|
      tag.strip!
      key = "#{Rails.configuration.redis['tag_prefix']}#{tag}"
      content = JSON.dump(meta)
      r.sadd?(key, content)
      puts("add tag #{tag} to redis")
      r.sadd?(set_key, tag)
    end #each
  end 

  def save_content_to_redis(r, file_name, content, tags)
    puts "add #{file_name}'s content to redis"
    key_prefix = Rails.configuration.redis['post_prefix']
    key = key_prefix + file_name
    r.set(key, JSON.dump({content: content, tags: tags})) 
  end

  def clear_all_tags(r)
    puts "Delete all the tags"
    pattern = "#{Rails.configuration.redis['tag_prefix']}*"
    r.keys(pattern).each do |key|
      puts "delete #{key}"
      r.del(key)
    end
  end

end
```

In a nutshell, the whole progress is 

- Delete the post  list set
- Clear the tag list set
- Delete all the individual  tag sets
- Scan all the files in the posts directory, if the file is a markdown file, then parse it
- Treat the first line as a json string and serialize it to a python dictionary, this is the meta info of the markdown file
- Save the rest of the file as the content of the post to redis, and use the file name as part of the key
- Generate the post list set
- Build the tag list set

### Test the paring result

Since I do not want to hard code all the redis keys in code, I simply put them in the redis config file.

```yaml
# config/redis.yml
production:
  host: todo
  port: todo
  post_set_key: live_posts
  post_prefix: live_p
  tag_prefix: live_t_
  tag_set_key: live_tags

development:
  host: localhost
  port: 6379
  post_set_key: dev_posts
  post_prefix: dev_p_
  tag_prefix: dev_t_
  tag_set_key: dev_tags
```

It’s easy to test the paring result via redis cli

```bash
redis-cli
# get all the items in post list set
zrange  dev_posts  0 -1
# display all the tags
SMEMBERS dev_tags
# show all the individual tag keys
keys dev_t_*
# get the content of an individual tag set, in this case the tag name is string
smembers dev_t_string
# get a content of a post, the file name of the post is first-post.md 
get dev_p_first-post
```

## Conclusion

I have finished the most significant part of the blog system, I would like to say that most of the work is already done. I can image your dismay because it seems that we have not started yet. We did not even create a rails controller. I understand, remind you the upcoming part is in the comfort zone of rails, it is kind of no brainer. In the next post let’s try to finish the typical stuff of a rails project.