export default function FeedList({ posts }) {
  return (
    <section className="feed">
      {posts.map((post) => (
        <article className="post" key={post.id}>
          <div className="post-header">
            <strong>{post.author}</strong>
            <span>{post.time}</span>
          </div>
          <p>{post.text}</p>
        </article>
      ))}
    </section>
  );
}
