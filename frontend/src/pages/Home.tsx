import { useEffect, useState } from 'react'
import { User } from '../../bindings/github.com/ha-li/Whip/internal/user'

function Home() {
  const [name, setName] = useState<string | null>(null);

  useEffect(() => {
    User.FirstName()
      .then(setName)
      .catch((err) => {
        console.error('Failed to read user name', err);
        setName('there');
      });
  }, []);

  return (
    <section className="home">
      <h1 className="home-greeting">
        Hello{name !== null && <span className="home-name"> {name}</span>}
      </h1>
    </section>
  )
}

export default Home
