import { useEffect, useState } from 'react'
import { Motd, type Quote } from '../../bindings/github.com/ha-li/Whip/internal/motd'
import { User } from '../../bindings/github.com/ha-li/Whip/internal/user'
import './Home.css'

function Home() {
  const [name, setName] = useState<string | null>(null);
  const [quote, setQuote] = useState<Quote | null>(null);

  useEffect(() => {
    User.FirstName()
      .then(setName)
      .catch((err) => {
        console.error('Failed to read user name', err);
        setName('there');
      });
  }, []);

  useEffect(() => {
    Motd.Today()
      .then(setQuote)
      .catch((err) => console.error('Failed to read the quote of the day', err));
  }, []);

  return (
    <section className="home">
      <h1 className="home-greeting">
        Hello{name !== null && <span className="home-name"> {name}</span>}
      </h1>
      {quote && (
        <figure className="home-quote">
          <blockquote className="home-quote-text">{quote.text}</blockquote>
          <figcaption className="home-quote-author">{quote.author}</figcaption>
        </figure>
      )}
    </section>
  )
}

export default Home
