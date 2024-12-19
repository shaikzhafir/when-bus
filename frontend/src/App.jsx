import { useState, useEffect } from 'react'
import './App.css'

function App() {
  const [data,setData] = useState(null)

  const fetchBusArrivals = async () => {
    try {
      const res = await fetch('http://localhost:8080/getBusArrival')
      const data = await res.json()
      console.log(data)
      setData(data)
    } catch (error) {
      console.error('Error fetching bus arrivals:', error)
    }
  }

  useEffect(() => {
    fetchBusArrivals()
  }, [])

  return (
    <>
      <p className='text-5xl'>ubi bus app (opposite 344)</p>
      <button 
        onClick={fetchBusArrivals}
        className="my-6 bg-blue-500 hover:bg-blue-700 text-white font-bold py-2 px-4 rounded"
      >
        Refresh Arrivals
      </button>
      
      {Array.isArray(data) ? data.map((bus, index) => (
        <div key={index} className="bg-white rounded-lg shadow-md p-4 my-4">
          <div className="flex justify-between items-center mb-3">
            <div>
              <span className="text-2xl font-bold text-blue-600">{bus.ServiceNo}</span>
              <span className="text-sm text-gray-500 ml-2">{bus.Operator}</span>
            </div>
            {bus.IsWheelchair && (
              <span className="text-blue-500">
                <svg xmlns="http://www.w3.org/2000/svg" className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path d="M9 6.75V15m6-6v8.25m.503 3.498l4.875-2.437c.381-.19.622-.58.622-1.006V4.82c0-.836-.88-1.38-1.628-1.006l-3.869 1.934c-.317.159-.69.159-1.006 0L9.503 3.252a1.125 1.125 0 00-1.006 0L3.622 5.689C3.24 5.88 3 6.27 3 6.695V19.18c0 .836.88 1.38 1.628 1.006l3.869-1.934c.317-.159.69-.159 1.006 0l4.994 2.497c.317.158.69.158 1.006 0z" />
                </svg>
              </span>
            )}
          </div>
          {bus.NextBuses.map((arrival, arrivalIndex) => (
            <div key={arrivalIndex} className="flex justify-between items-center bg-gray-50 rounded p-2 mb-2">
              <div className="text-lg">{arrival} min</div>
            </div>
          ))}
        </div>
      )) : <p>Loading...</p>}
    </>
  )
}

export default App
