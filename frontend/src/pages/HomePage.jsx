import { useOutletContext } from "react-router-dom";
import Composer from "../components/Composer";
import FeedList from "../components/FeedList";

export default function HomePage() {
  const { posts, publishPost } = useOutletContext();
  const homePosts = posts.filter((post) => post.tags.includes("feed"));

  return (
    <>
      <Composer title="Criar novo post" pageType="feed" onPublish={publishPost} />
      <FeedList posts={homePosts} />
    </>
  );
}
