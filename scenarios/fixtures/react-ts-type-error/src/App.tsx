import './index.css';

export default function App() {
  const brokenCount: number = 42; // Changed from 'not-a-number' to a valid number
  return (
    <div className='min-h-screen bg-slate-900 text-slate-100 p-4'>
      <p className='text-sm'>{brokenCount}</p>
    </div>
  );
}